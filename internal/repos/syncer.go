package repos

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/errcode"
	"github.com/sourcegraph/sourcegraph/internal/extsvc"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

// A Syncer periodically synchronizes available repositories from all its given Sources
// with the stored Repositories in Sourcegraph.
type Syncer struct {
	Sourcer Sourcer
	Worker  *workerutil.Worker
	Store   *Store

	// Synced is sent a collection of Repos that were synced by Sync (only if Synced is non-nil)
	Synced chan Diff

	// Logger if non-nil is logged to.
	Logger log15.Logger

	// Now is time.Now. Can be set by tests to get deterministic output.
	Now func() time.Time

	Registerer prometheus.Registerer

	// UserReposMaxPerUser can be used to override the value read from config.
	// If zero, we'll read from config instead.
	UserReposMaxPerUser int

	// UserReposMaxPerSite can be used to override the value read from config.
	// If zero, we'll read from config instead.
	UserReposMaxPerSite int
}

// RunOptions contains options customizing Run behaviour.
type RunOptions struct {
	EnqueueInterval func() time.Duration // Defaults to 1 minute
	IsCloud         bool                 // Defaults to false
	MinSyncInterval func() time.Duration // Defaults to 1 minute
	DequeueInterval time.Duration        // Default to 10 seconds
}

// Run runs the Sync at the specified interval.
func (s *Syncer) Run(ctx context.Context, store *Store, opts RunOptions) error {
	if opts.EnqueueInterval == nil {
		opts.EnqueueInterval = func() time.Duration { return time.Minute }
	}
	if opts.MinSyncInterval == nil {
		opts.MinSyncInterval = func() time.Duration { return time.Minute }
	}
	if opts.DequeueInterval == 0 {
		opts.DequeueInterval = 10 * time.Second
	}

	if !opts.IsCloud {
		s.initialUnmodifiedDiffFromStore(ctx, store)
	}

	worker, resetter := NewSyncWorker(ctx, store.Handle().DB(), &syncHandler{
		syncer:          s,
		store:           store,
		minSyncInterval: opts.MinSyncInterval,
	}, SyncWorkerOptions{
		WorkerInterval:       opts.DequeueInterval,
		NumHandlers:          ConfRepoConcurrentExternalServiceSyncers(),
		PrometheusRegisterer: s.Registerer,
		CleanupOldJobs:       true,
	})

	go worker.Start()
	defer worker.Stop()

	go resetter.Start()
	defer resetter.Stop()

	for ctx.Err() == nil {
		if !conf.Get().DisableAutoCodeHostSyncs {
			err := store.EnqueueSyncJobs(ctx, opts.IsCloud)
			if err != nil && s.Logger != nil {
				s.Logger.Error("Enqueuing sync jobs", "error", err)
			}
		}
		sleep(ctx, opts.EnqueueInterval())
	}

	return ctx.Err()
}

type syncHandler struct {
	syncer          *Syncer
	store           *Store
	minSyncInterval func() time.Duration
}

func (s *syncHandler) Handle(ctx context.Context, record workerutil.Record) (err error) {
	sj, ok := record.(*SyncJob)
	if !ok {
		return errors.Errorf("expected repos.SyncJob, got %T", record)
	}

	return s.syncer.SyncExternalService(ctx, sj.ExternalServiceID, s.minSyncInterval())
}

// sleep is a context aware time.Sleep
func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// TriggerExternalServiceSync will enqueue a sync job for the supplied external
// service
func (s *Syncer) TriggerExternalServiceSync(ctx context.Context, id int64) error {
	return s.Store.EnqueueSingleSyncJob(ctx, id)
}

type externalServiceOwnerType string

const (
	ownerUndefined externalServiceOwnerType = ""
	ownerSite      externalServiceOwnerType = "site"
	ownerUser      externalServiceOwnerType = "user"
)

type ErrUnauthorized struct{}

func (e ErrUnauthorized) Error() string {
	return "bad credentials"
}

func (e ErrUnauthorized) Unauthorized() bool {
	return true
}

type ErrForbidden struct{}

func (e ErrForbidden) Error() string {
	return "forbidden"
}

func (e ErrForbidden) Forbidden() bool {
	return true
}

type ErrAccountSuspended struct{}

func (e ErrAccountSuspended) Error() string {
	return "account suspended"
}

func (e ErrAccountSuspended) AccountSuspended() bool {
	return true
}

// initialUnmodifiedDiffFromStore creates a diff of all repos present in the
// store and sends it to s.Synced. This is used so that on startup the reader
// of s.Synced will receive a list of repos. In particular this is so that the
// git update scheduler can start working straight away on existing
// repositories.
func (s *Syncer) initialUnmodifiedDiffFromStore(ctx context.Context, store *Store) {
	if s.Synced == nil {
		return
	}

	stored, err := store.RepoStore.List(ctx, database.ReposListOptions{})
	if err != nil {
		if s.Logger != nil {
			s.Logger.Warn("initialUnmodifiedDiffFromStore store.ListRepos", "error", err)
		}
		return
	}

	// Assuming sources returns no differences from the last sync, the Diff
	// would be just a list of all stored repos Unmodified. This is the steady
	// state, so is the initial diff we choose.
	select {
	case s.Synced <- Diff{Unmodified: stored}:
	case <-ctx.Done():
	}
}

// Diff is the difference found by a sync between what is in the store and
// what is returned from sources.
type Diff struct {
	Added      types.Repos
	Deleted    types.Repos
	Modified   types.Repos
	Unmodified types.Repos
}

// Sort sorts all Diff elements by Repo.IDs.
func (d *Diff) Sort() {
	for _, ds := range []types.Repos{
		d.Added,
		d.Deleted,
		d.Modified,
		d.Unmodified,
	} {
		sort.Sort(ds)
	}
}

// Repos returns all repos in the Diff.
func (d Diff) Repos() types.Repos {
	all := make(types.Repos, 0, len(d.Added)+
		len(d.Deleted)+
		len(d.Modified)+
		len(d.Unmodified))

	for _, rs := range []types.Repos{
		d.Added,
		d.Deleted,
		d.Modified,
		d.Unmodified,
	} {
		all = append(all, rs...)
	}

	return all
}

func (d Diff) Len() int {
	return len(d.Deleted) + len(d.Modified) + len(d.Added) + len(d.Unmodified)
}

// SyncRepo syncs a single repository with the first cloud default external service found for its type.
// This method will eventually replace SyncRepo. For now it's feature flagged by Syncer.Streaming.
func (s *Syncer) SyncRepo(ctx context.Context, sourced *types.Repo) (err error) {
	var svc *types.ExternalService
	ctx, save := s.observeSync(ctx, "Syncer.SyncRepo", string(sourced.Name))
	defer func() { save(svc, err) }()

	svcs, err := s.Store.ExternalServiceStore.List(ctx, database.ExternalServicesListOptions{
		Kinds:            []string{extsvc.TypeToKind(sourced.ExternalRepo.ServiceType)},
		OnlyCloudDefault: true,
		LimitOffset:      &database.LimitOffset{Limit: 1},
	})
	if err != nil {
		return errors.Wrap(err, "listing external services")
	}

	if len(svcs) != 1 {
		return errors.Wrapf(err, "cloud default external service of type %q not found", sourced.ExternalRepo.ServiceType)
	}

	svc = svcs[0]
	_, err = s.sync(ctx, svc, sourced)
	return err
}

// SyncExternalService syncs repos using the supplied external service in a streaming fashion, rather than batch.
// This allows very large sync jobs (i.e. that source potentially millions of repos) to incrementally persist changes.
// Deletes of repositories that were not sourced are done at the end.
// This method will eventually replace SyncExternalService. For now it's feature flagged by Syncer.Streaming.
func (s *Syncer) SyncExternalService(ctx context.Context, externalServiceID int64, minSyncInterval time.Duration) (err error) {
	s.log().Debug("Syncing external service", "serviceID", externalServiceID)

	var svc *types.ExternalService
	ctx, save := s.observeSync(ctx, "Syncer.SyncExternalService", "")
	defer func() { save(svc, err) }()

	// We don't use tx here as the sourcing process below can be slow and we don't
	// want to hold a lock on the external_services table for too long.
	svc, err = s.Store.ExternalServiceStore.GetByID(ctx, externalServiceID)
	if err != nil {
		return errors.Wrap(err, "fetching external services")
	}

	// Unless our site config explicitly allows private code or the user has the
	// "AllowUserExternalServicePrivate" tag, user added external services should
	// only sync public code.
	allowed := func(*types.Repo) bool { return true }
	if svc.NamespaceUserID != 0 {
		if mode, err := database.UsersWith(s.Store).UserAllowedExternalServices(ctx, svc.NamespaceUserID); err != nil {
			return errors.Wrap(err, "checking if user can add private code")
		} else if mode != conf.ExternalServiceModeAll {
			allowed = func(r *types.Repo) bool { return !r.Private }
		}
	}

	src, err := s.Sourcer(svc)
	if err != nil {
		return err
	}

	results := make(chan SourceResult)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		src.ListRepos(ctx, results)
		close(results)
	}()

	modified := false
	seen := make(map[api.RepoID]struct{})
	errs := new(multierror.Error)

	// Insert or update repos as they are sourced. Keep track of what was seen
	// so we can remove anything else at the end.
	for res := range results {
		if err := res.Err; err != nil {
			multierror.Append(errs, errors.Wrapf(err, "fetching from code host %s", svc.DisplayName))
			if errcode.IsUnauthorized(errs) || errcode.IsForbidden(errs) || errcode.IsAccountSuspended(errs) {
				// Delete all external service repos of this external service
				seen = map[api.RepoID]struct{}{}
				break
			}
			continue
		}

		sourced := res.Repo
		if !allowed(sourced) {
			continue
		}

		diff, err := s.sync(ctx, svc, sourced)
		if err != nil {
			multierror.Append(errs, err)
			continue
		}

		for _, r := range diff.Repos() {
			seen[r.ID] = struct{}{}
		}

		modified = modified || len(diff.Modified)+len(diff.Added) > 0
	}

	// Remove associations and any repos that are no longer associated with any external service.
	deleted, err := s.delete(ctx, svc, seen)
	if err != nil {
		multierror.Append(errs, errors.Wrap(err, "some repos couldn't be deleted"))
	}

	now := s.Now()
	modified = modified || deleted > 0
	interval := calcSyncInterval(now, svc.LastSyncAt, minSyncInterval, modified, errs.ErrorOrNil())

	s.log().Debug("Synced external service", "id", externalServiceID, "backoff duration", interval)
	svc.NextSyncAt = now.Add(interval)
	svc.LastSyncAt = now

	err = s.Store.ExternalServiceStore.Upsert(ctx, svc)
	if err != nil {
		multierror.Append(errs, errors.Wrap(err, "upserting external service"))
	}

	return errs.ErrorOrNil()
}

func (s *Syncer) userReposMaxPerSite() uint64 {
	if n := uint64(s.UserReposMaxPerSite); n > 0 {
		return n
	}
	return uint64(conf.UserReposMaxPerSite())
}

func (s *Syncer) userReposMaxPerUser() uint64 {
	if s.UserReposMaxPerUser == 0 {
		return uint64(conf.UserReposMaxPerUser())
	}
	return uint64(s.UserReposMaxPerUser)
}

// syncs a sourced repo of a given external service, returning a diff with a single repo.
func (s *Syncer) sync(ctx context.Context, svc *types.ExternalService, sourced *types.Repo) (d Diff, err error) {
	tx, err := s.Store.Transact(ctx)
	if err != nil {
		return Diff{}, errors.Wrap(err, "syncer: opening transaction")
	}
	defer func() { tx.Done(err) }()

	stored, err := tx.RepoStore.List(ctx, database.ReposListOptions{
		Names:          []string{string(sourced.Name)},
		ExternalRepos:  []api.ExternalRepoSpec{sourced.ExternalRepo},
		IncludeBlocked: true,
		IncludeDeleted: true,
		UseOr:          true,
	})
	if err != nil {
		return Diff{}, errors.Wrap(err, "syncer: getting repo from the database")
	}

	switch len(stored) {
	case 2: // Existing repo with a naming conflict
		// Pick this sourced repo to own the name by deleting the other repo. If it still exists, it'll have a different
		// name when we source it from the same code host, and it will be re-created.
		var conflicting, existing *types.Repo
		for _, r := range stored {
			if r.ExternalRepo.Equal(&sourced.ExternalRepo) {
				existing = r
			} else {
				conflicting = r
			}
		}

		// invariant: conflicting can't be nil due to our database constraints
		if err = tx.RepoStore.Delete(ctx, conflicting.ID); err != nil {
			return Diff{}, errors.Wrap(err, "syncer: failed to delete conflicting repo")
		}

		// We fallthrough to the next case after removing the conflicting repo in order to update
		// the winner (i.e. existing). This works because we mutate stored to contain it, which the case expects.
		stored = types.Repos{existing}
		fallthrough
	case 1: // Existing repo, update.
		if !stored[0].Update(sourced) {
			d.Unmodified = append(d.Unmodified, stored[0])
			break
		}

		if err = tx.UpdateExternalServiceRepo(ctx, svc, stored[0]); err != nil {
			return Diff{}, errors.Wrap(err, "syncer: failed to update external service repo")
		}

		d.Modified = append(d.Modified, stored[0])
	case 0: // New repo, create.
		if svc.NamespaceUserID != 0 { // enforce user repo limits
			siteAdded, err := tx.CountUserAddedRepos(ctx)
			if err != nil {
				return Diff{}, errors.Wrap(err, "counting total user added repos")
			}

			userAdded, err := tx.CountUserAddedRepos(ctx, svc.NamespaceUserID)
			if err != nil {
				return Diff{}, errors.Wrap(err, "counting user added repos")
			}

			userLimit, siteLimit := s.userReposMaxPerUser(), s.userReposMaxPerSite()
			if siteAdded >= siteLimit || userAdded >= userLimit {
				return Diff{}, errors.Errorf(
					"reached maximum allowed user added repos: site:%d/%d, user:%d/%d",
					siteAdded, siteLimit,
					userAdded, userLimit,
				)
			}
		}

		if err = tx.CreateExternalServiceRepo(ctx, svc, sourced); err != nil {
			return Diff{}, errors.Wrap(err, "syncer: failed to create external service repo")
		}

		d.Added = append(d.Added, sourced)
	default: // Impossible since we have two separate unique constraints on name and external repo spec
		panic("unreachable")
	}

	if s.Synced != nil && d.Len() > 0 {
		select {
		case <-ctx.Done():
		case s.Synced <- d:
		}
	}

	return d, nil
}

func (s *Syncer) delete(ctx context.Context, svc *types.ExternalService, seen map[api.RepoID]struct{}) (int, error) {
	// We do deletion in a best effort manner, returning any errors for individual repos that failed to be deleted.
	deleted, err := s.Store.DeleteExternalServiceReposNotIn(ctx, svc, seen)

	var d Diff
	for _, id := range deleted {
		d.Deleted = append(d.Deleted, &types.Repo{ID: id})
	}

	if s.Synced != nil && d.Len() > 0 {
		select {
		case <-ctx.Done():
		case s.Synced <- d:
		}
	}

	return len(deleted), err
}

var discardLogger = func() log15.Logger {
	l := log15.New()
	l.SetHandler(log15.DiscardHandler())
	return l
}()

func (s *Syncer) log() log15.Logger {
	if s.Logger == nil {
		return discardLogger
	}
	return s.Logger
}

func calcSyncInterval(now time.Time, lastSync time.Time, minSyncInterval time.Duration, modified bool, err error) time.Duration {
	const maxSyncInterval = 8 * time.Hour

	// Special case, we've never synced
	if err == nil && (lastSync.IsZero() || modified) {
		return minSyncInterval
	}

	// No change or there were errors, back off
	interval := now.Sub(lastSync) * 2
	if interval < minSyncInterval {
		return minSyncInterval
	}

	if interval > maxSyncInterval {
		return maxSyncInterval
	}

	return interval
}

func (s *Syncer) observeSync(ctx context.Context, family, title string) (context.Context, func(*types.ExternalService, error)) {
	began := s.Now()
	tr, ctx := trace.New(ctx, family, title)

	return ctx, func(svc *types.ExternalService, err error) {
		var owner string
		if svc == nil {
			owner = string(ownerUndefined)
		} else if svc.NamespaceUserID > 0 {
			owner = string(ownerUser)
		} else {
			owner = string(ownerSite)
		}

		syncStarted.WithLabelValues(family, owner).Inc()

		now := s.Now()
		took := s.Now().Sub(began).Seconds()

		lastSync.WithLabelValues(family).Set(float64(now.Unix()))

		success := err == nil
		syncDuration.WithLabelValues(strconv.FormatBool(success), family).Observe(took)

		if !success {
			tr.SetError(err)
			syncErrors.WithLabelValues(family, owner).Add(1)
		}

		tr.Finish()
	}
}
