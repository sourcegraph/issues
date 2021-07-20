package repos

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
	"github.com/inconshreveable/log15"
	otlog "github.com/opentracing/opentracing-go/log"
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

	// SingleRepoSynced is sent the result of a single repo sync that were synced by SyncRepo (only if
	// SingleRepoSynced is non-nil)
	SingleRepoSynced chan Diff

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

	// Streaming, if true, will make the Syncer use the streaming implementations of
	// SyncExternalService and SyncRepo.
	Streaming bool
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

	tx := s.store
	if !s.syncer.Streaming {
		tx, err = s.store.Transact(ctx)
		if err != nil {
			return err
		}
		defer func() { err = tx.Done(err) }()
	}

	return s.syncer.SyncExternalService(ctx, tx, sj.ExternalServiceID, s.minSyncInterval())
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

// SyncExternalService syncs repos using the supplied external service.
func (s *Syncer) SyncExternalService(ctx context.Context, tx *Store, externalServiceID int64, minSyncInterval time.Duration) (err error) {
	if s.Streaming {
		return s.StreamingSyncExternalService(ctx, tx, externalServiceID, minSyncInterval)
	}

	var (
		diff             Diff
		unauthorized     bool
		accountSuspended bool
		forbidden        bool
	)

	if s.Logger != nil {
		s.Logger.Debug("Syncing external service", "serviceID", externalServiceID)
	}

	owner := ownerUndefined
	ctx, save := s.observe(ctx, "Syncer.SyncExternalService", "")
	defer save(&diff, &owner, &err)

	ids := []int64{externalServiceID}
	// We don't use tx here as the sourcing process below can be slow and we don't
	// want to hold a lock on the external_services table for too long.
	svcs, err := s.Store.ExternalServiceStore.List(ctx, database.ExternalServicesListOptions{IDs: ids})
	if err != nil {
		return errors.Wrap(err, "fetching external services")
	}

	if len(svcs) != 1 {
		return errors.Errorf("want 1 external service but got %d", len(svcs))
	}
	svc := svcs[0]

	if svc.NamespaceUserID > 0 {
		owner = ownerUser
	} else {
		owner = ownerSite
	}

	onSourced := func(*types.Repo) error { return nil } // noop

	if owner == ownerUser {
		// If we are over our limit for user added repos we abort the sync
		totalAllowed := uint64(s.UserReposMaxPerSite)
		if totalAllowed == 0 {
			totalAllowed = uint64(conf.UserReposMaxPerSite())
		}
		userAdded, err := tx.CountUserAddedRepos(ctx)
		if err != nil {
			return errors.Wrap(err, "counting user added repos")
		}
		if userAdded >= totalAllowed {
			return errors.Errorf("reached maximum allowed user added repos: %d", userAdded)
		}

		// If this is a user owned external service we won't stream our inserts as we limit the number allowed.
		// Instead, we'll track the number of sourced repos and if we exceed our limit we'll bail out.
		var sourcedRepoCount int64
		maxAllowed := s.UserReposMaxPerUser
		if maxAllowed == 0 {
			maxAllowed = conf.UserReposMaxPerUser()
		}
		onSourced = func(r *types.Repo) error {
			newCount := atomic.AddInt64(&sourcedRepoCount, 1)
			if newCount >= int64(maxAllowed) {
				return errors.Errorf("per user repo count has exceeded allowed limit: %d", maxAllowed)
			}
			return nil
		}
	} else if s.SingleRepoSynced != nil {
		// This is a site level external service. We have a channel to handle streaming inserts,
		// therefore we should create an inserter. Note that it inserts outside of our transaction
		// so that repos are visible to the rest of our system immediately.
		onSourced, err = s.makeNewRepoInserter(ctx, s.Store, owner)
		if err != nil {
			return errors.Wrap(err, "syncer.sync.streaming")
		}
	}

	// Fetch repos from the source
	var sourced types.Repos
	if sourced, err = s.sourced(ctx, svc, onSourced); err != nil {
		unauthorized = errcode.IsUnauthorized(err)
		forbidden = errcode.IsForbidden(err)
		accountSuspended = errcode.IsAccountSuspended(err)

		// As a special case, if we fail due to bad credentials or account suspension we
		// should behave as if zero repos were found. This is so that revoked tokens
		// cause repos to be removed correctly.
		if !unauthorized && !accountSuspended && !forbidden {
			return errors.Wrap(err, "fetching from code host "+svc.DisplayName)
		}
		log15.Warn("Non fatal error during sync", "externalService", svc.ID, "unauthorized", unauthorized, "accountSuspended", accountSuspended, "forbidden", forbidden)
	}

	// Unless our site config explicitly allows private code or the user has the
	// "AllowUserExternalServicePrivate" tag, user added external services should
	// only sync public code.
	if owner == ownerUser {
		if mode, err := database.UsersWith(tx).UserAllowedExternalServices(ctx, svc.NamespaceUserID); err != nil {
			return errors.Wrap(err, "checking if user can add private code")
		} else if mode != conf.ExternalServiceModeAll {
			sourced = sourced.Filter(func(r *types.Repo) bool { return !r.Private })
		}
	}

	var storedServiceRepos types.Repos
	// Fetch repos from our DB related to externalServiceID
	if storedServiceRepos, err = tx.RepoStore.List(ctx, database.ReposListOptions{ExternalServiceIDs: []int64{externalServiceID}}); err != nil {
		return errors.Wrap(err, "syncer.sync.store.list-repos")
	}

	// Now fetch any possible name conflicts.
	// Repo names must be globally unique, if there's conflict we need to deterministically choose one.
	var conflicting types.Repos
	if len(sourced) > 0 {
		if conflicting, err = tx.RepoStore.List(ctx, database.ReposListOptions{Names: sourced.Names()}); err != nil {
			return errors.Wrap(err, "syncer.sync.store.list-repos")
		}
		conflicting = conflicting.Filter(func(r *types.Repo) bool {
			for _, id := range r.ExternalServiceIDs() {
				if id == externalServiceID {
					return false
				}
			}

			return true
		})
	}

	// Add the conflicts to the list of repos fetched from the database.
	// NewDiff modifies the storedServiceRepos slice so we clone it before passing it
	storedServiceReposAndConflicting := append(storedServiceRepos.Clone(), conflicting...)

	// Our stored repo could have multiple sources in its Sources map. Our sourced repo will only every have
	// one repo in its Sources map. In order for our diff code to operate we should add the other sources to
	// the sourced repo.
	storedByURI := make(map[string]*types.Repo, len(storedServiceRepos))
	for _, r := range storedServiceRepos {
		storedByURI[r.URI] = r
	}
	sourcedByURI := make(map[string]*types.Repo, len(sourced))
	for _, r := range sourced {
		sourcedByURI[r.URI] = r
	}
	for _, r := range sourced {
		stored, ok := storedByURI[r.URI]
		if !ok {
			continue
		}
		for urn, source := range stored.Sources {
			if _, exists := r.Sources[urn]; exists {
				// Don't replace, only add
				continue
			}
			r.Sources[urn] = source
		}
	}

	// Find the diff associated with only the currently syncing external service.
	diff = newDiff(svc, sourced, storedServiceRepos)
	resolveNameConflicts(&diff, conflicting)
	upserts := s.upserts(diff)

	// Delete from external_service_repos only. Deletes need to happen first so that we don't end up with
	// constraint violations later.
	sdiff := s.sourcesUpserts(&diff, storedServiceReposAndConflicting)
	if err = tx.UpsertSources(ctx, nil, nil, sdiff.Deleted); err != nil {
		return errors.Wrap(err, "syncer.sync.store.delete-sources")
	}

	// Next, insert or modify existing repos. This is needed so that the next call
	// to UpsertSources has valid repo ids
	if err = tx.UpsertRepos(ctx, upserts...); err != nil {
		return errors.Wrap(err, "syncer.sync.store.upsert-repos")
	}

	// Only modify added and modified relationships in external_service_repos, deleted was
	// handled above
	// Recalculate sdiff so that we have foreign keys
	sdiff = s.sourcesUpserts(&diff, storedServiceReposAndConflicting)
	if err = tx.UpsertSources(ctx, sdiff.Added, sdiff.Modified, nil); err != nil {
		return errors.Wrap(err, "syncer.sync.store.upsert-sources")
	}

	now := s.Now()
	interval := calcSyncInterval(now, svc.LastSyncAt, minSyncInterval, diff)
	if s.Logger != nil {
		s.Logger.Debug("Synced external service", "id", externalServiceID, "backoff duration", interval)
	}
	svc.NextSyncAt = now.Add(interval)
	svc.LastSyncAt = now

	err = tx.ExternalServiceStore.Upsert(ctx, svc)
	if err != nil {
		return errors.Wrap(err, "upserting external service")
	}

	if s.Synced != nil {
		select {
		case s.Synced <- diff:
		case <-ctx.Done():
		}
	}

	if unauthorized {
		return &ErrUnauthorized{}
	}
	if forbidden {
		return &ErrForbidden{}
	}
	if accountSuspended {
		return &ErrAccountSuspended{}
	}

	return nil
}

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

// We need to resolve name conflicts by deciding whether to keep the newly added repo
// or the repo that already exists in the database.
// If the new repo wins, then the old repo is added to the diff.Deleted slice.
// If the old repo wins, then the new repo is no longer inserted and is filtered out from
// the diff.Added slice.
func resolveNameConflicts(diff *Diff, conflicting types.Repos) {
	var toDelete types.Repos
	diff.Added = diff.Added.Filter(func(r *types.Repo) bool {
		for _, cr := range conflicting {
			if cr.Name == r.Name {
				// The repos are conflicting, we deterministically choose the one
				// that has the smallest external repo spec.
				switch cr.ExternalRepo.Compare(r.ExternalRepo) {
				case -1:
					// the repo that is currently existing in the database wins
					// causing the new one to be filtered out
					return false
				case 1:
					// the new repo wins so the old repo is deleted along with all of its relationships.
					toDelete = append(toDelete, cr.With(func(r *types.Repo) { r.Sources = nil }))
				}

				return true
			}
		}

		return true
	})
	diff.Modified = diff.Modified.Filter(func(r *types.Repo) bool {
		for _, cr := range conflicting {
			if cr.Name == r.Name {
				// The repos are conflicting, we deterministically choose the one
				// that has the smallest external repo spec.
				switch cr.ExternalRepo.Compare(r.ExternalRepo) {
				case -1:
					// the repo that is currently existing in the database wins
					// causing the new one to be filtered out
					toDelete = append(toDelete, r.With(func(r *types.Repo) { r.Sources = nil }))
					return false
				case 1:
					// the new repo wins so the old repo is deleted along with all of its relationships.
					toDelete = append(toDelete, cr.With(func(r *types.Repo) { r.Sources = nil }))
				}

				return true
			}
		}

		return true
	})
	diff.Deleted = append(diff.Deleted, toDelete...)
}

func calcSyncInterval(now time.Time, lastSync time.Time, minSyncInterval time.Duration, diff Diff) time.Duration {
	const maxSyncInterval = 8 * time.Hour

	// Special case, we've never synced
	if lastSync.IsZero() {
		return minSyncInterval
	}

	// If there is any change, sync again shortly
	if len(diff.Added) > 0 || len(diff.Deleted) > 0 || len(diff.Modified) > 0 {
		return minSyncInterval
	}

	// No change, back off
	interval := now.Sub(lastSync) * 2
	if interval < minSyncInterval {
		return minSyncInterval
	}
	if interval > maxSyncInterval {
		return maxSyncInterval
	}
	return interval
}

// SyncRepo runs the syncer on a single repository.
func (s *Syncer) SyncRepo(ctx context.Context, store *Store, sourcedRepo *types.Repo) (err error) {
	if s.Streaming {
		return s.StreamingSyncRepo(ctx, sourcedRepo)
	}

	var diff Diff

	// SyncRepo is only used for site level external services on sourcegraph.com
	owner := ownerSite

	ctx, save := s.observe(ctx, "Syncer.SyncRepo", string(sourcedRepo.Name))
	defer save(&diff, &owner, &err)

	var txs *Store
	if txs, err = store.Transact(ctx); err != nil {
		return errors.Wrap(err, "Syncer.SyncRepo.transact")
	}
	defer txs.Done(err)
	store = txs

	diff, err = s.syncRepo(ctx, store, false, true, sourcedRepo)
	return err
}

// insertIfNew is a specialization of SyncRepo. It will insert sourcedRepo
// if there are no related repositories, otherwise does nothing.
func (s *Syncer) insertIfNew(ctx context.Context, store *Store, sourcedRepo *types.Repo, owner externalServiceOwnerType) (err error) {
	var diff Diff

	ctx, save := s.observe(ctx, "Syncer.InsertIfNew", string(sourcedRepo.Name))
	defer save(&diff, &owner, &err)

	// insertIfNew is only used for streaming inserter, which is currently only enabled on customer
	// instances. Therefore we set publicOnly to false because customer instances do not have any
	// limitation for private code.
	diff, err = s.syncRepo(ctx, store, true, false, sourcedRepo)
	return err
}

// syncRepo syncs a single repo that has been sourced from a single external service.
func (s *Syncer) syncRepo(ctx context.Context, store *Store, insertOnly bool, publicOnly bool, sourcedRepo *types.Repo) (diff Diff, err error) {
	if publicOnly && sourcedRepo.Private {
		return Diff{}, nil
	}

	var storedRepos types.Repos
	args := database.ReposListOptions{
		Names:         []string{string(sourcedRepo.Name)},
		ExternalRepos: []api.ExternalRepoSpec{sourcedRepo.ExternalRepo},
		UseOr:         true,
	}
	if storedRepos, err = store.RepoStore.List(ctx, args); err != nil {
		return Diff{}, errors.Wrap(err, "syncer.syncrepo.store.list-repos")
	}

	if insertOnly && len(storedRepos) > 0 {
		return Diff{}, nil
	}

	// sourcedRepo only knows about one source so we need to add in the remaining
	// stored sources
	if len(storedRepos) == 1 {
		for k, v := range storedRepos[0].Sources {
			// Don't update the source from sourcedRepo
			if _, ok := sourcedRepo.Sources[k]; ok {
				continue
			}
			sourcedRepo.Sources[k] = v
		}
	}

	// NewDiff modifies the stored slice so we clone it before passing it
	storedCopy := storedRepos.Clone()

	diff = NewDiff([]*types.Repo{sourcedRepo}, storedRepos)

	// We trust that if we determine that a repo needs to be deleted it should be deleted
	// from all external services. By setting sources to nil this is forced when we call
	// UpsertSources below.
	for i := range diff.Deleted {
		diff.Deleted[i].Sources = nil
	}

	// Delete from external_service_repos only. Deletes need to happen first so that we don't end up with
	// constraint violations later.
	sdiff := s.sourcesUpserts(&diff, storedCopy)
	if err = store.UpsertSources(ctx, nil, nil, sdiff.Deleted); err != nil {
		return Diff{}, errors.Wrap(err, "syncer.syncrepo.store.delete-sources")
	}

	// Next, insert or modify existing repos. This is needed so that the next call
	// to UpsertSources has valid repo ids
	upserts := s.upserts(diff)
	if err = store.UpsertRepos(ctx, upserts...); err != nil {
		return Diff{}, errors.Wrap(err, "syncer.syncrepo.store.upsert-repos")
	}

	// Only modify added and modified relationships in external_service_repos, deleted was
	// handled above.
	// Recalculate sdiff so that we have foreign keys
	sdiff = s.sourcesUpserts(&diff, storedCopy)
	if err = store.UpsertSources(ctx, sdiff.Added, sdiff.Modified, nil); err != nil {
		return Diff{}, errors.Wrap(err, "syncer.syncrepo.store.upsert-sources")
	}

	if s.SingleRepoSynced != nil {
		select {
		case s.SingleRepoSynced <- diff:
		case <-ctx.Done():
		}
	}

	return diff, nil
}

// upserts returns a slice containing modified or added repos from a Diff. Deleted
// repos are ignored.
func (s *Syncer) upserts(diff Diff) []*types.Repo {
	now := s.Now()
	upserts := make([]*types.Repo, 0, len(diff.Added)+len(diff.Modified))

	for _, repo := range diff.Modified {
		repo.UpdatedAt, repo.DeletedAt = now, time.Time{}
		upserts = append(upserts, repo)
	}

	for _, repo := range diff.Added {
		repo.CreatedAt, repo.UpdatedAt, repo.DeletedAt = now, now, time.Time{}
		upserts = append(upserts, repo)
	}

	return upserts
}

type sourceDiff struct {
	Added, Modified, Deleted map[api.RepoID][]types.SourceInfo
}

// sourcesUpserts creates a diff for sources based on the repositoried diff.
func (s *Syncer) sourcesUpserts(diff *Diff, stored []*types.Repo) *sourceDiff {
	sdiff := sourceDiff{
		Added:    make(map[api.RepoID][]types.SourceInfo),
		Modified: make(map[api.RepoID][]types.SourceInfo),
		Deleted:  make(map[api.RepoID][]types.SourceInfo),
	}

	// When a repository is added, add its sources map to the list
	// of sourceInfos
	for _, repo := range diff.Added {
		for _, si := range repo.Sources {
			sdiff.Added[repo.ID] = append(sdiff.Added[repo.ID], *si)
		}
	}

	// When a repository is modified, check if its source map
	// has been modified, and if so compute the diff.
	for _, repo := range diff.Modified {
		if repo.Sources == nil {
			continue
		}

		for _, storedRepo := range stored {
			if storedRepo.ID == repo.ID {
				s.sourceDiff(repo.ID, &sdiff, storedRepo.Sources, repo.Sources)
				break
			}
		}
	}

	// When a repository is deleted, check if its source map
	// has been modified, and if so compute the diff.
	for _, repo := range diff.Deleted {
		for _, storedRepo := range stored {
			if storedRepo.ID == repo.ID {
				s.sourceDiff(repo.ID, &sdiff, storedRepo.Sources, repo.Sources)
				break
			}
		}
	}

	return &sdiff
}

// sourceDiff computes the diff between the oldSources and the newSources,
// and updates the Added, Modified and Deleted in place of `diff`.
func (s *Syncer) sourceDiff(repoID api.RepoID, diff *sourceDiff, oldSources, newSources map[string]*types.SourceInfo) {
	for k, oldSrc := range oldSources {
		if newSrc, ok := newSources[k]; ok {
			if oldSrc.CloneURL != newSrc.CloneURL {
				// The source has been modified
				diff.Modified[repoID] = append(diff.Modified[repoID], *newSrc)
			}

			continue
		}

		diff.Deleted[repoID] = append(diff.Deleted[repoID], *oldSrc)
	}

	for k := range newSources {
		if _, ok := oldSources[k]; ok {
			continue
		}

		diff.Added[repoID] = append(diff.Added[repoID], *newSources[k])
	}
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

// NewDiff returns a diff from the given sourced and stored repos.
func NewDiff(sourced, stored []*types.Repo) (diff Diff) {
	return newDiff(nil, sourced, stored)
}

func newDiff(svc *types.ExternalService, sourced, stored []*types.Repo) (diff Diff) {
	// Sort sourced so we merge deterministically
	sort.Sort(types.Repos(sourced))

	byID := make(map[api.ExternalRepoSpec]*types.Repo, len(sourced))
	for _, r := range sourced {
		if old := byID[r.ExternalRepo]; old != nil {
			merge(old, r)
		} else {
			byID[r.ExternalRepo] = r
		}
	}

	// Ensure names are unique case-insensitively. We don't merge when finding
	// a conflict on name, we deterministically pick which sourced repo to
	// keep. Can't merge since they represent different repositories
	// (different external ID).
	byName := make(map[string]*types.Repo, len(byID))
	for _, r := range byID {
		k := strings.ToLower(string(r.Name))
		if old := byName[k]; old == nil {
			byName[k] = r
		} else {
			keep, discard := pick(r, old)
			byName[k] = keep
			delete(byID, discard.ExternalRepo)
		}
	}

	seenID := make(map[api.ExternalRepoSpec]bool, len(stored))

	for _, old := range stored {
		src := byID[old.ExternalRepo]

		// if the repo hasn't been found in the sourced repo list
		// we add it to the Deleted slice and, if the service is provided
		// we remove the service from its source map.
		if src == nil {
			if svc != nil {
				if _, ok := old.Sources[svc.URN()]; ok {
					old = old.Clone()
					delete(old.Sources, svc.URN())
				}
			}

			diff.Deleted = append(diff.Deleted, old)
		} else if old.Update(src) {
			diff.Modified = append(diff.Modified, old)
		} else {
			diff.Unmodified = append(diff.Unmodified, old)
		}

		seenID[old.ExternalRepo] = true
	}

	for _, r := range byID {
		if !seenID[r.ExternalRepo] {
			diff.Added = append(diff.Added, r)
		}
	}

	return diff
}

func merge(o, n *types.Repo) {
	for id, src := range o.Sources {
		n.Sources[id] = src
	}
	o.Update(n)
}

func (s *Syncer) sourced(ctx context.Context, svc *types.ExternalService, onSourced ...func(*types.Repo) error) ([]*types.Repo, error) {
	srcs, err := s.Sourcer(svc)
	if err != nil {
		return nil, err
	}

	return listAll(ctx, srcs, onSourced...)
}

// makeNewRepoInserter returns a function that will insert repos.
// If publicOnly is set it will never insert a private repo.
func (s *Syncer) makeNewRepoInserter(ctx context.Context, store *Store, owner externalServiceOwnerType) (func(*types.Repo) error, error) {
	// insertIfNew requires querying the store for related repositories, and
	// will do nothing if `insertOnly` is set and there are any related repositories. Most
	// repositories will already have related repos, so to avoid that cost we
	// ask the store for all repositories and only do syncRepo if it might
	// be an insert.
	ids, err := store.ListExternalRepoSpecs(ctx)
	if err != nil {
		return nil, err
	}

	return func(r *types.Repo) error {
		// We know this won't be an insert.
		if _, ok := ids[r.ExternalRepo]; ok {
			return nil
		}

		err := s.insertIfNew(ctx, store, r, owner)
		if err != nil && s.Logger != nil {
			// Best-effort, final syncer will handle this repo if this failed.
			s.Logger.Warn("streaming insert failed", "external_id", r.ExternalRepo, "error", err)
			return err
		}
		return nil
	}, nil
}

func (s *Syncer) observe(ctx context.Context, family, title string) (context.Context, func(*Diff, *externalServiceOwnerType, *error)) {
	began := s.Now()
	tr, ctx := trace.New(ctx, family, title)

	return ctx, func(d *Diff, owner *externalServiceOwnerType, err *error) {
		var ownerTag string
		if owner != nil {
			ownerTag = string(*owner)
		}

		syncStarted.WithLabelValues(family, ownerTag).Inc()

		now := s.Now()
		took := s.Now().Sub(began).Seconds()

		fields := make([]otlog.Field, 0, 7)
		for state, repos := range map[string]types.Repos{
			"added":      d.Added,
			"modified":   d.Modified,
			"deleted":    d.Deleted,
			"unmodified": d.Unmodified,
		} {
			fields = append(fields, otlog.Int(state+".count", len(repos)))
			if state != "unmodified" {
				fields = append(fields,
					otlog.Object(state+".repos", repos.Names()))

				if len(repos) > 0 && s.Logger != nil {
					s.Logger.Debug(family, "diff."+state, repos.NamesSummary())
					s.Logger.Debug(family, "diff."+state, repos.NamesSummary())
				}
			}
			syncedTotal.WithLabelValues(state, family).Add(float64(len(repos)))
		}

		tr.LogFields(fields...)

		lastSync.WithLabelValues(family).Set(float64(now.Unix()))

		success := err == nil || *err == nil
		syncDuration.WithLabelValues(strconv.FormatBool(success), family).Observe(took)

		if !success {
			tr.SetError(*err)
			syncErrors.WithLabelValues(family, ownerTag).Add(1)
		}

		tr.Finish()
	}
}

// StreamingSyncRepo syncs a single repository with the first cloud default external service found for its type.
// This method will eventually replace SyncRepo. For now it's feature flagged by Syncer.Streaming.
func (s *Syncer) StreamingSyncRepo(ctx context.Context, sourced *types.Repo) (err error) {
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
	_, err = s.sync(ctx, s.Store, svc, sourced)
	return err
}

// StreamingSyncExternalService syncs repos using the supplied external service in a streaming fashion, rather than batch.
// This allows very large sync jobs (i.e. that source potentially millions of repos) to incrementally persist changes.
// Deletes of repositories that were not sourced are done at the end.
// This method will eventually replace SyncExternalService. For now it's feature flagged by Syncer.Streaming.
func (s *Syncer) StreamingSyncExternalService(ctx context.Context, tx *Store, externalServiceID int64, minSyncInterval time.Duration) (err error) {
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
		if mode, err := database.UsersWith(tx).UserAllowedExternalServices(ctx, svc.NamespaceUserID); err != nil {
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

		diff, err := s.sync(ctx, tx, svc, sourced)
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
	deleted, err := s.delete(ctx, tx, svc, seen)
	if err != nil {
		multierror.Append(errs, errors.Wrap(err, "some repos couldn't be deleted"))
	}

	now := s.Now()
	modified = modified || deleted > 0
	interval := calcStreamingSyncInterval(now, svc.LastSyncAt, minSyncInterval, modified, errs.ErrorOrNil())

	s.log().Debug("Synced external service", "id", externalServiceID, "backoff duration", interval)
	svc.NextSyncAt = now.Add(interval)
	svc.LastSyncAt = now

	err = tx.ExternalServiceStore.Upsert(ctx, svc)
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
func (s *Syncer) sync(ctx context.Context, tx *Store, svc *types.ExternalService, sourced *types.Repo) (d Diff, err error) {
	if !tx.InTransaction() {
		tx, err = tx.Transact(ctx)
		if err != nil {
			return Diff{}, errors.Wrap(err, "syncer: opening transaction")
		}
		defer func() { tx.Done(err) }()
	}

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

func (s *Syncer) delete(ctx context.Context, tx *Store, svc *types.ExternalService, seen map[api.RepoID]struct{}) (int, error) {
	// We do deletion in a best effort manner, returning any errors for individual repos that failed to be deleted.
	deleted, err := tx.DeleteExternalServiceReposNotIn(ctx, svc, seen)

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

func calcStreamingSyncInterval(now time.Time, lastSync time.Time, minSyncInterval time.Duration, modified bool, err error) time.Duration {
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
