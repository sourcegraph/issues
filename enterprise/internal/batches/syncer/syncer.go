package syncer

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/sources"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/state"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/batches/store"
	btypes "github.com/sourcegraph/sourcegraph/enterprise/internal/batches/types"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/httpcli"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

// SyncRegistry manages a changesetSyncer per code host
type SyncRegistry struct {
	ctx         context.Context
	syncStore   SyncStore
	httpFactory *httpcli.Factory

	// Used to receive high priority sync requests
	priorityNotify chan []int64

	mu sync.Mutex
	// key is normalized code host url, also called external_service_id on the repo table
	syncers map[string]*changesetSyncer
}

type SyncStore interface {
	ListCodeHosts(ctx context.Context, opts store.ListCodeHostsOpts) ([]*btypes.CodeHost, error)
	ListChangesetSyncData(context.Context, store.ListChangesetSyncDataOpts) ([]*btypes.ChangesetSyncData, error)
	GetChangeset(context.Context, store.GetChangesetOpts) (*btypes.Changeset, error)
	UpdateChangesetCodeHostState(ctx context.Context, cs *btypes.Changeset) error
	UpsertChangesetEvents(ctx context.Context, cs ...*btypes.ChangesetEvent) error
	GetSiteCredential(ctx context.Context, opts store.GetSiteCredentialOpts) (*btypes.SiteCredential, error)
	Transact(context.Context) (*store.Store, error)
	Repos() *database.RepoStore
	ExternalServices() *database.ExternalServiceStore
	Clock() func() time.Time
	DB() dbutil.DB
	GetExternalServiceIDs(ctx context.Context, opts store.GetExternalServiceIDsOpts) ([]int64, error)
	UserCredentials() *database.UserCredentialsStore
}

// NewSyncRegistry creates a new sync registry which starts a syncer for each code host and will update them
// when external services are changed, added or removed.
func NewSyncRegistry(ctx context.Context, cstore SyncStore, cf *httpcli.Factory) *SyncRegistry {
	r := &SyncRegistry{
		ctx:            ctx,
		syncStore:      cstore,
		httpFactory:    cf,
		priorityNotify: make(chan []int64, 500),
		syncers:        make(map[string]*changesetSyncer),
	}

	if err := r.syncCodeHosts(ctx); err != nil {
		log15.Error("Fetching initial list of code hosts", "err", err)
	}

	go r.handlePriorityItems()

	return r
}

// Add adds a syncer for the code host associated with the supplied code host if the syncer hasn't
// already been added and starts it.
func (s *SyncRegistry) Add(codeHost *btypes.CodeHost) {
	// This should never happen since the store does the filtering for us, but let's be super duper extra cautious.
	if !codeHost.IsSupported() {
		log15.Info("Code host not support by batch changes", "type", codeHost.ExternalServiceType, "url", codeHost.ExternalServiceID)
		return
	}

	syncerKey := codeHost.ExternalServiceID

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.syncers[syncerKey]; ok {
		// Already added
		return
	}

	// We need to be able to cancel the syncer if the code host is removed
	ctx, cancel := context.WithCancel(s.ctx)

	syncer := &changesetSyncer{
		syncStore:      s.syncStore,
		httpFactory:    s.httpFactory,
		codeHostURL:    syncerKey,
		cancel:         cancel,
		priorityNotify: make(chan []int64, 500),
	}

	s.syncers[syncerKey] = syncer

	go syncer.Run(ctx)
}

// EnqueueChangesetSyncs will enqueue the changesets with the supplied ids for high priority syncing.
// An error indicates that no changesets have been enqueued.
func (s *SyncRegistry) EnqueueChangesetSyncs(ctx context.Context, ids []int64) error {
	// The channel below is buffered so we'll usually send without blocking.
	// It is important not to block here as this method is called from the UI
	select {
	case s.priorityNotify <- ids:
	default:
		return errors.New("high priority sync capacity reached")
	}
	return nil
}

// HandleExternalServiceSync handles changes to external services.
func (s *SyncRegistry) HandleExternalServiceSync(es api.ExternalService) {
	if btypes.IsKindSupported(es.Kind) {
		if err := s.syncCodeHosts(s.ctx); err != nil {
			log15.Error("Syncing on change of code hosts", "err", err)
		}
	}
}

// handlePriorityItems fetches changesets in the priority queue from the database and passes them
// to the appropriate syncer.
func (s *SyncRegistry) handlePriorityItems() {
	fetchSyncData := func(ids []int64) ([]*btypes.ChangesetSyncData, error) {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		return s.syncStore.ListChangesetSyncData(ctx, store.ListChangesetSyncDataOpts{ChangesetIDs: ids})
	}
	for {
		select {
		case <-s.ctx.Done():
			return
		case ids := <-s.priorityNotify:
			syncData, err := fetchSyncData(ids)
			if err != nil {
				log15.Error("Fetching sync data", "err", err)
				continue
			}

			// Group changesets by code host
			changesetByHost := make(map[string][]int64)
			for _, d := range syncData {
				changesetByHost[d.RepoExternalServiceID] = append(changesetByHost[d.RepoExternalServiceID], d.ChangesetID)
			}

			// Anonymous func so we can use defer
			func() {
				s.mu.Lock()
				defer s.mu.Unlock()
				for host, changesets := range changesetByHost {
					syncer, ok := s.syncers[host]
					if !ok {
						continue
					}

					select {
					case syncer.priorityNotify <- changesets:
					default:
					}
				}
			}()
		}
	}
}

// syncCodeHosts fetches the list of currently active code hosts on the Sourcegraph instance.
// The running syncers will then be matched against those and missing ones are spawned and
// excess ones are stopped.
func (s *SyncRegistry) syncCodeHosts(ctx context.Context) error {
	codeHosts, err := s.syncStore.ListCodeHosts(ctx, store.ListCodeHostsOpts{})
	if err != nil {
		return err
	}

	codeHostsByExternalServiceID := make(map[string]*btypes.CodeHost)

	// Add and start syncers
	for _, host := range codeHosts {
		codeHostsByExternalServiceID[host.ExternalServiceID] = host
		s.Add(host)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Clean up old syncers.
	for syncerKey := range s.syncers {
		// If there is no code host for the syncer anymore, we want to stop it.
		if _, ok := codeHostsByExternalServiceID[syncerKey]; !ok {
			syncer, exists := s.syncers[syncerKey]
			if exists {
				delete(s.syncers, syncerKey)
				syncer.cancel()
			}
		}
	}
	return nil
}

// A changesetSyncer periodically syncs metadata of changesets
// saved in the database.
type changesetSyncer struct {
	syncStore   SyncStore
	httpFactory *httpcli.Factory

	codeHostURL string

	// scheduleInterval determines how often a new schedule will be computed.
	// NOTE: It involves a DB query but no communication with code hosts.
	scheduleInterval time.Duration

	queue          *changesetPriorityQueue
	priorityNotify chan []int64

	// Replaceable for testing
	syncFunc func(ctx context.Context, id int64) error

	// cancel should be called to stop this syncer
	cancel context.CancelFunc
}

var syncerMetrics = struct {
	syncs                   *prometheus.CounterVec
	priorityQueued          *prometheus.CounterVec
	syncDuration            *prometheus.HistogramVec
	computeScheduleDuration *prometheus.HistogramVec
	scheduleSize            *prometheus.GaugeVec
	behindSchedule          *prometheus.GaugeVec
}{}

func init() {
	syncerMetrics.syncs = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "src_repoupdater_changeset_syncer_syncs",
		Help: "Total number of changeset syncs",
	}, []string{"codehost", "success"})
	syncerMetrics.priorityQueued = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "src_repoupdater_changeset_syncer_priority_queued",
		Help: "Total number of priority items added to queue",
	}, []string{"codehost"})
	syncerMetrics.syncDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "src_repoupdater_changeset_syncer_sync_duration_seconds",
		Help:    "Time spent syncing changesets",
		Buckets: []float64{1, 2, 5, 10, 30, 60, 120},
	}, []string{"codehost", "success"})
	syncerMetrics.computeScheduleDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "src_repoupdater_changeset_syncer_compute_schedule_duration_seconds",
		Help:    "Time spent computing changeset schedule",
		Buckets: []float64{1, 2, 5, 10, 30, 60, 120},
	}, []string{"codehost", "success"})
	syncerMetrics.scheduleSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "src_repoupdater_changeset_syncer_schedule_size",
		Help: "The number of changesets scheduled to sync",
	}, []string{"codehost"})
	syncerMetrics.behindSchedule = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "src_repoupdater_changeset_syncer_behind_schedule",
		Help: "The number of changesets behind schedule",
	}, []string{"codehost"})
}

// Run will start the process of changeset syncing. It is long running
// and is expected to be launched once at startup.
func (s *changesetSyncer) Run(ctx context.Context) {
	log15.Debug("Starting changeset syncer", "codeHostURL", s.codeHostURL)
	scheduleInterval := s.scheduleInterval
	if scheduleInterval == 0 {
		scheduleInterval = 2 * time.Minute
	}
	if s.syncFunc == nil {
		s.syncFunc = s.SyncChangeset
	}
	s.queue = newChangesetPriorityQueue()
	// How often to refresh the schedule
	scheduleTicker := time.NewTicker(scheduleInterval)

	if !conf.Get().DisableAutoCodeHostSyncs {
		// Get initial schedule
		if sched, err := s.computeSchedule(ctx); err != nil {
			// Non fatal as we'll try again later in the main loop
			log15.Error("Computing schedule", "err", err)
		} else {
			s.queue.Upsert(sched...)
		}
	}

	var next scheduledSync
	var ok bool

	// NOTE: All mutations of the queue should be done is this loop as operations on the queue
	// are not safe for concurrent use
	for {
		var timer *time.Timer
		var timerChan <-chan time.Time
		next, ok = s.queue.Peek()

		if ok {
			// Queue isn't empty
			if next.priority == priorityHigh {
				// Fire ASAP
				timer = time.NewTimer(0)
			} else {
				// Use scheduled time
				timer = time.NewTimer(time.Until(next.nextSync))
			}
			timerChan = timer.C
		}

		select {
		case <-ctx.Done():
			return
		case <-scheduleTicker.C:
			if timer != nil {
				timer.Stop()
			}

			if conf.Get().DisableAutoCodeHostSyncs {
				continue
			}

			start := s.syncStore.Clock()()
			schedule, err := s.computeSchedule(ctx)
			labelValues := []string{s.codeHostURL, strconv.FormatBool(err == nil)}
			syncerMetrics.computeScheduleDuration.WithLabelValues(labelValues...).Observe(s.syncStore.Clock()().Sub(start).Seconds())
			if err != nil {
				log15.Error("Computing queue", "err", err)
				continue
			}
			syncerMetrics.scheduleSize.WithLabelValues(s.codeHostURL).Set(float64(len(schedule)))
			s.queue.Upsert(schedule...)
			var behindSchedule int
			now := s.syncStore.Clock()()
			for _, ss := range schedule {
				if ss.nextSync.Before(now) {
					behindSchedule++
				}
			}
			syncerMetrics.behindSchedule.WithLabelValues(s.codeHostURL).Set(float64(behindSchedule))
		case <-timerChan:
			start := s.syncStore.Clock()()
			err := s.syncFunc(ctx, next.changesetID)
			labelValues := []string{s.codeHostURL, strconv.FormatBool(err == nil)}
			syncerMetrics.syncDuration.WithLabelValues(labelValues...).Observe(s.syncStore.Clock()().Sub(start).Seconds())
			syncerMetrics.syncs.WithLabelValues(labelValues...).Add(1)

			if err != nil {
				log15.Error("Syncing changeset", "err", err)
				// We'll continue and remove it as it'll get retried on next schedule
			}

			// Remove item now that it has been processed
			s.queue.Remove(next.changesetID)
			syncerMetrics.scheduleSize.WithLabelValues(s.codeHostURL).Dec()
		case ids := <-s.priorityNotify:
			if timer != nil {
				timer.Stop()
			}
			for _, id := range ids {
				item, ok := s.queue.Get(id)
				if !ok {
					// Item has been recently synced and removed or we have an invalid id
					// We have no way of telling the difference without making a DB call so
					// add a new item anyway which will just lead to a harmless error later
					item = scheduledSync{
						changesetID: id,
						nextSync:    time.Time{},
					}
				}
				item.priority = priorityHigh
				s.queue.Upsert(item)
				syncerMetrics.scheduleSize.WithLabelValues(s.codeHostURL).Inc()
			}
			syncerMetrics.priorityQueued.WithLabelValues(s.codeHostURL).Add(float64(len(ids)))
		}
	}
}

func (s *changesetSyncer) computeSchedule(ctx context.Context) ([]scheduledSync, error) {
	syncData, err := s.syncStore.ListChangesetSyncData(ctx, store.ListChangesetSyncDataOpts{ExternalServiceID: s.codeHostURL})
	if err != nil {
		return nil, errors.Wrap(err, "listing changeset sync data")
	}

	ss := make([]scheduledSync, len(syncData))
	for i := range syncData {
		nextSync := NextSync(s.syncStore.Clock(), syncData[i])

		ss[i] = scheduledSync{
			changesetID: syncData[i].ChangesetID,
			nextSync:    nextSync,
		}
	}

	return ss, nil
}

// SyncChangeset will sync a single changeset given its id.
func (s *changesetSyncer) SyncChangeset(ctx context.Context, id int64) error {
	log15.Debug("SyncChangeset", "syncer", s.codeHostURL, "id", id)

	cs, err := s.syncStore.GetChangeset(ctx, store.GetChangesetOpts{
		ID: id,

		// Enforce precondition given in changeset sync state query.
		ReconcilerState:  btypes.ReconcilerStateCompleted,
		PublicationState: btypes.ChangesetPublicationStatePublished,
	})
	if err != nil {
		if err == store.ErrNoResults {
			log15.Debug("SyncChangeset not found", "id", id)
			return nil
		}
		return err
	}

	repo, err := s.syncStore.Repos().Get(ctx, cs.RepoID)
	if err != nil {
		return err
	}

	source, err := loadChangesetSource(ctx, s.httpFactory, s.syncStore, repo)
	if err != nil {
		return err
	}

	return SyncChangeset(ctx, s.syncStore, source, repo, cs)
}

// SyncChangeset refreshes the metadata of the given changeset and
// updates them in the database.
func SyncChangeset(ctx context.Context, syncStore SyncStore, source sources.ChangesetSource, repo *types.Repo, c *btypes.Changeset) (err error) {
	repoChangeset := &sources.Changeset{Repo: repo, Changeset: c}
	if err := source.LoadChangeset(ctx, repoChangeset); err != nil {
		if !errors.HasType(err, sources.ChangesetNotFoundError{}) {
			// Store the error as the syncer error.
			errMsg := err.Error()
			c.SyncErrorMessage = &errMsg
			if err2 := syncStore.UpdateChangesetCodeHostState(ctx, c); err2 != nil {
				return errors.Wrap(err, err2.Error())
			}
			return err
		}

		if !c.IsDeleted() {
			c.SetDeleted()
		}
	}

	events, err := c.Events()
	if err != nil {
		return err
	}
	state.SetDerivedState(ctx, syncStore.Repos(), c, events)

	tx, err := syncStore.Transact(ctx)
	if err != nil {
		return err
	}
	defer func() { err = tx.Done(err) }()

	// Reset syncer error message state.
	c.SyncErrorMessage = nil

	err = tx.UpdateChangesetCodeHostState(ctx, c)
	if err != nil {
		return err
	}

	return tx.UpsertChangesetEvents(ctx, events...)
}

func loadChangesetSource(ctx context.Context, cf *httpcli.Factory, syncStore SyncStore, repo *types.Repo) (sources.ChangesetSource, error) {
	srcer := sources.NewSourcer(cf)
	// This is a ChangesetSource authenticated with the external service
	// token.
	source, err := srcer.ForRepo(ctx, syncStore, repo)
	if err != nil {
		return nil, err
	}
	// Try to use a site credential. If none is present, this falls back to
	// the external service config. This code path should error in the future.
	return sources.WithSiteAuthenticator(ctx, syncStore, source, repo)
}
