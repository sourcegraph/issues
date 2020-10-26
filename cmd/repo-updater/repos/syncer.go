package repos

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/inconshreveable/log15"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

// A Syncer periodically synchronizes available repositories from all its given Sources
// with the stored Repositories in Sourcegraph.
type Syncer struct {
	Sourcer Sourcer
	Worker  *workerutil.Worker
	Store   Store

	// Synced is sent a collection of Repos that were synced by Sync (only if Synced is non-nil)
	Synced chan Diff

	// SubsetSynced is sent the result of a single repo sync that were synced by SyncRepo (only if SubsetSynced is non-nil)
	SubsetSynced chan Diff

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

	// syncErrors contains the last error returned by the Sourcer during each
	// sync per external service. It's reset with each service sync and if the sync produced no error, it's
	// set to nil.
	syncErrors   map[int64]error
	syncErrorsMu sync.Mutex

	enqueueSignal signal
}

// RunOptions contains options customizing Run behaviour.
type RunOptions struct {
	EnqueueInterval func() time.Duration // Defaults to 1 minute
	IsCloud         bool                 // Defaults to false
	MinSyncInterval func() time.Duration // Defaults to 1 minute
	DequeueInterval time.Duration        // Default to 10 seconds
}

// Run runs the Sync at the specified interval.
func (s *Syncer) Run(pctx context.Context, db *sql.DB, store Store, opts RunOptions) error {
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
		s.initialUnmodifiedDiffFromStore(pctx, store)
	}

	worker, resetter := NewSyncWorker(pctx, db, &syncHandler{
		db:              db,
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

	for pctx.Err() == nil {
		ctx, cancel := contextWithSignalCancel(pctx, s.enqueueSignal.Watch())

		if err := store.EnqueueSyncJobs(ctx, opts.IsCloud); err != nil && s.Logger != nil {
			s.Logger.Error("Enqueuing sync jobs", "error", err)
		}

		sleep(ctx, opts.EnqueueInterval())

		cancel()
	}

	return pctx.Err()
}

type syncHandler struct {
	db              *sql.DB
	syncer          *Syncer
	store           Store
	minSyncInterval func() time.Duration
}

func (s *syncHandler) Handle(ctx context.Context, tx dbworkerstore.Store, record workerutil.Record) (err error) {
	sj, ok := record.(*SyncJob)
	if !ok {
		return fmt.Errorf("expected repos.SyncJob, got %T", record)
	}

	store := s.store
	if ws, ok := s.store.(WithStore); ok {
		store = ws.With(tx.Handle().DB())
	}

	return s.syncer.SyncExternalService(ctx, store, sj.ExternalServiceID, s.minSyncInterval())
}

// contextWithSignalCancel will return a context which will be cancelled if
// signal fires. Callers need to call cancel when done.
func contextWithSignalCancel(ctx context.Context, signal <-chan struct{}) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		select {
		case <-ctx.Done():
		case <-signal:
			cancel()
		}
	}()

	return ctx, cancel
}

// sleep is a context aware time.Sleep
func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// TriggerEnqueueSyncJobs will enqueue any pending sync jobs now.
func (s *Syncer) TriggerEnqueueSyncJobs() {
	s.enqueueSignal.Trigger()
}

// SyncExternalService syncs repos using the supplied external service.
func (s *Syncer) SyncExternalService(ctx context.Context, tx Store, externalServiceID int64, minSyncInterval time.Duration) (err error) {
	var diff Diff

	if s.Logger != nil {
		s.Logger.Debug("Syncing external service", "serviceID", externalServiceID)
	}

	ctx, save := s.observe(ctx, "Syncer.SyncExternalService", "")
	defer save(&diff, &err)
	defer s.setOrResetLastSyncErr(externalServiceID, &err)

	ids := []int64{externalServiceID}
	svcs, err := tx.ListExternalServices(ctx, StoreListExternalServicesArgs{IDs: ids})
	if err != nil {
		return errors.Wrap(err, "fetching external services")
	}

	if len(svcs) != 1 {
		return errors.Errorf("want 1 external service but got %d", len(svcs))
	}
	svc := svcs[0]
	isUserOwned := svc.NamespaceUserID > 0

	onSourced := func(*Repo) error { return nil } //noop

	if isUserOwned {
		// If we are over our limit for user added repos we abort the sync
		totalAllowed := uint64(s.UserReposMaxPerSite)
		if totalAllowed == 0 {
			totalAllowed = uint64(ConfUserReposMaxPerSite())
		}
		userAdded, err := tx.CountUserAddedRepos(ctx)
		if err != nil {
			return errors.Wrap(err, "counting user added repos")
		}
		if userAdded >= totalAllowed {
			return fmt.Errorf("reached maximum allowed user added repos: %d", userAdded)
		}

		// If this is a user owned external service we won't stream our inserts as we limit the number allowed.
		// Instead, we'll track the number of sourced repos and if we exceed our limit we'll bail out.
		var sourcedRepoCount int64
		maxAllowed := s.UserReposMaxPerUser
		if maxAllowed == 0 {
			maxAllowed = ConfUserReposMaxPerUser()
		}
		onSourced = func(r *Repo) error {
			newCount := atomic.AddInt64(&sourcedRepoCount, 1)
			if newCount >= int64(maxAllowed) {
				return fmt.Errorf("per user repo count has exceeded allowed limit: %d", maxAllowed)
			}
			return nil
		}
	} else if s.SubsetSynced != nil {
		// This is a site admin owned external service so we should stream inserts ASAP.
		// It should insert outside of our transaction so that repos are visible to the rest of our
		// system immediately.
		onSourced, err = s.makeNewRepoInserter(ctx, s.Store, isUserOwned)
		if err != nil {
			return errors.Wrap(err, "syncer.sync.streaming")
		}
	}

	// Fetch repos from the source
	var sourced Repos
	if sourced, err = s.sourced(ctx, svcs, onSourced); err != nil {
		return errors.Wrap(err, "syncer.sync.sourced")
	}

	// User added external services should only sync public code
	if isUserOwned {
		sourced = sourced.Filter(func(r *Repo) bool { return !r.Private })
	}

	var storedServiceRepos Repos
	// Fetch repos from our DB related to externalServiceID
	if storedServiceRepos, err = tx.ListRepos(ctx, StoreListReposArgs{ExternalServiceID: externalServiceID}); err != nil {
		return errors.Wrap(err, "syncer.sync.store.list-repos")
	}

	// Now fetch any possible name conflicts.
	// Repo names must be globally unique, if there's conflict we need to deterministically choose one.
	var conflicting Repos
	if conflicting, err = tx.ListRepos(ctx, StoreListReposArgs{Names: sourced.Names()}); err != nil {
		return errors.Wrap(err, "syncer.sync.store.list-repos")
	}
	conflicting = conflicting.Filter(func(r *Repo) bool {
		for _, id := range r.ExternalServiceIDs() {
			if id == externalServiceID {
				return false
			}
		}

		return true
	})

	// Add the conflicts to the list of repos fetched from the db.
	// NewDiff modifies the storedServiceRepos slice so we clone it before passing it
	storedServiceReposAndConflicting := append(storedServiceRepos.Clone(), conflicting...)

	// Our stored repo could have multiple sources in its Sources map. Our sourced repo will only every have
	// one repo in its Sources map. In order for our diff code to operate we should add the other sources to
	// the sourced repo.
	storedByURI := make(map[string]*Repo, len(storedServiceRepos))
	for _, r := range storedServiceRepos {
		storedByURI[r.URI] = r
	}
	sourcedByURI := make(map[string]*Repo, len(sourced))
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
	// The trigger 'trig_soft_delete_orphan_repo_by_external_service_repo' will run
	// and remove any repos that no longer have any rows in the external_service_repos
	// table.
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

	err = tx.UpsertExternalServices(ctx, svc)
	if err != nil {
		return errors.Wrap(err, "upserting external service")
	}

	if s.Synced != nil {
		select {
		case s.Synced <- diff:
		case <-ctx.Done():
		}
	}

	return nil
}

// We need to resolve name conflicts by deciding whether to keep the newly added repo
// or the repo that already exists in the db.
// If the new repo wins, then the old repo is added to the diff.Deleted slice.
// If the old repo wins, then the new repo is no longer inserted and is filtered out from
// the diff.Added slice.
func resolveNameConflicts(diff *Diff, conflicting Repos) {
	var toDelete Repos
	diff.Added = diff.Added.Filter(func(r *Repo) bool {
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
					toDelete = append(toDelete, cr.With(func(r *Repo) { r.Sources = nil }))
				}

				return true
			}
		}

		return true
	})
	diff.Modified = diff.Modified.Filter(func(r *Repo) bool {
		for _, cr := range conflicting {
			if cr.Name == r.Name {
				// The repos are conflicting, we deterministically choose the one
				// that has the smallest external repo spec.
				switch cr.ExternalRepo.Compare(r.ExternalRepo) {
				case -1:
					// the repo that is currently existing in the database wins
					// causing the new one to be filtered out
					toDelete = append(toDelete, r.With(func(r *Repo) { r.Sources = nil }))
					return false
				case 1:
					// the new repo wins so the old repo is deleted along with all of its relationships.
					toDelete = append(toDelete, cr.With(func(r *Repo) { r.Sources = nil }))
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
func (s *Syncer) SyncRepo(ctx context.Context, store Store, sourcedRepo *Repo) (err error) {
	var diff Diff

	ctx, save := s.observe(ctx, "Syncer.SyncRepo", sourcedRepo.Name)
	defer save(&diff, &err)

	if tr, ok := store.(Transactor); ok {
		var txs TxStore
		if txs, err = tr.Transact(ctx); err != nil {
			return errors.Wrap(err, "Syncer.SyncRepo.transact")
		}
		defer txs.Done(err)
		store = txs
	}

	diff, err = s.syncRepo(ctx, store, false, true, sourcedRepo)
	return err
}

// insertIfNew is a specialization of SyncRepo. It will insert sourcedRepo
// if there are no related repositories, otherwise does nothing.
func (s *Syncer) insertIfNew(ctx context.Context, store Store, publicOnly bool, sourcedRepo *Repo) (err error) {
	var diff Diff

	ctx, save := s.observe(ctx, "Syncer.InsertIfNew", sourcedRepo.Name)
	defer save(&diff, &err)

	diff, err = s.syncRepo(ctx, store, true, publicOnly, sourcedRepo)
	return err
}

func (s *Syncer) syncRepo(ctx context.Context, store Store, insertOnly bool, publicOnly bool, sourcedRepo *Repo) (diff Diff, err error) {
	if publicOnly && sourcedRepo.Private {
		return Diff{}, nil
	}

	var storedSubset Repos
	args := StoreListReposArgs{
		Names:         []string{sourcedRepo.Name},
		ExternalRepos: []api.ExternalRepoSpec{sourcedRepo.ExternalRepo},
		UseOr:         true,
	}
	if storedSubset, err = store.ListRepos(ctx, args); err != nil {
		return Diff{}, errors.Wrap(err, "syncer.syncrepo.store.list-repos")
	}

	if insertOnly && len(storedSubset) > 0 {
		return Diff{}, nil
	}

	// NewDiff modifies the stored slice so we clone it before passing it
	storedCopy := storedSubset.Clone()

	diff = NewDiff([]*Repo{sourcedRepo}, storedSubset)

	// We trust that if we determine that a repo needs to be deleted it should be deleted
	// from all external services. By setting sources to nil this is forced when we call
	// UpsertSources below.
	for i := range diff.Deleted {
		diff.Deleted[i].Sources = nil
	}

	// Delete from external_service_repos only. Deletes need to happen first so that we don't end up with
	// constraint violations later.
	// The trigger 'trig_soft_delete_orphan_repo_by_external_service_repo' will run
	// and remove any repos that no longer have any rows in the external_service_repos
	// table.
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

	if s.SubsetSynced != nil {
		select {
		case s.SubsetSynced <- diff:
		case <-ctx.Done():
		}
	}

	return diff, nil
}

func (s *Syncer) upserts(diff Diff) []*Repo {
	now := s.Now()
	upserts := make([]*Repo, 0, len(diff.Added)+len(diff.Modified))

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
	Added, Modified, Deleted map[api.RepoID][]SourceInfo
}

// sourcesUpserts creates a diff for sources based on the repositoried diff.
func (s *Syncer) sourcesUpserts(diff *Diff, stored []*Repo) *sourceDiff {
	sdiff := sourceDiff{
		Added:    make(map[api.RepoID][]SourceInfo),
		Modified: make(map[api.RepoID][]SourceInfo),
		Deleted:  make(map[api.RepoID][]SourceInfo),
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
func (s *Syncer) sourceDiff(repoID api.RepoID, diff *sourceDiff, oldSources, newSources map[string]*SourceInfo) {
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
func (s *Syncer) initialUnmodifiedDiffFromStore(ctx context.Context, store Store) {
	if s.Synced == nil {
		return
	}

	stored, err := store.ListRepos(ctx, StoreListReposArgs{})
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
	Added      Repos
	Deleted    Repos
	Modified   Repos
	Unmodified Repos
}

// Sort sorts all Diff elements by Repo.IDs.
func (d *Diff) Sort() {
	for _, ds := range []Repos{
		d.Added,
		d.Deleted,
		d.Modified,
		d.Unmodified,
	} {
		sort.Sort(ds)
	}
}

// Repos returns all repos in the Diff.
func (d Diff) Repos() Repos {
	all := make(Repos, 0, len(d.Added)+
		len(d.Deleted)+
		len(d.Modified)+
		len(d.Unmodified))

	for _, rs := range []Repos{
		d.Added,
		d.Deleted,
		d.Modified,
		d.Unmodified,
	} {
		all = append(all, rs...)
	}

	return all
}

// NewDiff returns a diff from the given sourced and stored repos.
func NewDiff(sourced, stored []*Repo) (diff Diff) {
	return newDiff(nil, sourced, stored)
}

func newDiff(svc *ExternalService, sourced, stored []*Repo) (diff Diff) {
	// Sort sourced so we merge deterministically
	sort.Sort(Repos(sourced))

	byID := make(map[api.ExternalRepoSpec]*Repo, len(sourced))
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
	byName := make(map[string]*Repo, len(byID))
	for _, r := range byID {
		k := strings.ToLower(r.Name)
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

func merge(o, n *Repo) {
	for id, src := range o.Sources {
		n.Sources[id] = src
	}
	o.Update(n)
}

func (s *Syncer) sourced(ctx context.Context, svcs []*ExternalService, onSourced ...func(*Repo) error) ([]*Repo, error) {
	srcs, err := s.Sourcer(svcs...)
	if err != nil {
		return nil, err
	}

	return listAll(ctx, srcs, onSourced...)
}

// makeNewRepoInserter returns a function that will insert repos.
// If publicOnly is set it will never insert a private repo.
func (s *Syncer) makeNewRepoInserter(ctx context.Context, store Store, publicOnly bool) (func(*Repo) error, error) {
	// insertIfNew requires querying the store for related repositories, and
	// will do nothing if `insertOnly` is set and there are any related repositories. Most
	// repositories will already have related repos, so to avoid that cost we
	// ask the store for all repositories and only do syncRepo if it might
	// be an insert.
	ids, err := store.ListExternalRepoSpecs(ctx)
	if err != nil {
		return nil, err
	}

	return func(r *Repo) error {
		// We know this won't be an insert.
		if _, ok := ids[r.ExternalRepo]; ok {
			return nil
		}

		err := s.insertIfNew(ctx, store, publicOnly, r)
		if err != nil && s.Logger != nil {
			// Best-effort, final syncer will handle this repo if this failed.
			s.Logger.Warn("streaming insert failed", "external_id", r.ExternalRepo, "error", err)
		}
		return nil
	}, nil
}

func (s *Syncer) setOrResetLastSyncErr(serviceID int64, perr *error) {
	var err error
	if perr != nil {
		err = *perr
	}

	s.syncErrorsMu.Lock()
	defer s.syncErrorsMu.Unlock()
	if s.syncErrors == nil {
		s.syncErrors = make(map[int64]error)
	}

	if err == nil {
		delete(s.syncErrors, serviceID)
		return
	}
	s.syncErrors[serviceID] = err
}

// SyncErrors returns all errors that was produced in the last Sync run per external sevice. If
// no error was produced, this returns an empty slice.
// Errors are sorted by external service id.
func (s *Syncer) SyncErrors() []error {
	s.syncErrorsMu.Lock()
	defer s.syncErrorsMu.Unlock()

	ids := make([]int, 0, len(s.syncErrors))

	for id := range s.syncErrors {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	sorted := make([]error, len(ids))
	for i, id := range ids {
		sorted[i] = s.syncErrors[int64(id)]
	}

	return sorted
}

func (s *Syncer) observe(ctx context.Context, family, title string) (context.Context, func(*Diff, *error)) {
	began := s.Now()
	tr, ctx := trace.New(ctx, family, title)

	return ctx, func(d *Diff, err *error) {
		syncStarted.WithLabelValues(family).Inc()

		now := s.Now()
		took := s.Now().Sub(began).Seconds()

		fields := make([]otlog.Field, 0, 7)
		for state, repos := range map[string]Repos{
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
			syncErrors.WithLabelValues(family).Add(1)
		}

		tr.Finish()
	}
}

type signal struct {
	once sync.Once
	c    chan struct{}
}

func (s *signal) init() {
	s.once.Do(func() {
		s.c = make(chan struct{}, 1)
	})
}

func (s *signal) Trigger() {
	s.init()
	select {
	case s.c <- struct{}{}:
	default:
	}
}

func (s *signal) Watch() <-chan struct{} {
	s.init()
	return s.c
}
