package repos

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/db/dbutil"
	"github.com/sourcegraph/sourcegraph/internal/logging"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
	"github.com/sourcegraph/sourcegraph/internal/trace"
)

// ObservedSource returns a decorator that wraps a Source
// with error logging, Prometheus metrics and tracing.
func ObservedSource(l logging.ErrorLogger, m SourceMetrics) func(Source) Source {
	return func(s Source) Source {
		return &observedSource{
			Source:  s,
			metrics: m,
			log:     l,
		}
	}
}

// An observedSource wraps another Source with error logging,
// Prometheus metrics and tracing.
type observedSource struct {
	Source
	metrics SourceMetrics
	log     logging.ErrorLogger
}

// SourceMetrics encapsulates the Prometheus metrics of a Source.
type SourceMetrics struct {
	ListRepos *metrics.OperationMetrics
}

// MustRegister registers all metrics in SourceMetrics in the given
// prometheus.Registerer. It panics in case of failure.
func (sm SourceMetrics) MustRegister(r prometheus.Registerer) {
	r.MustRegister(sm.ListRepos.Count)
	r.MustRegister(sm.ListRepos.Duration)
	r.MustRegister(sm.ListRepos.Errors)
}

// NewSourceMetrics returns SourceMetrics that need to be registered
// in a Prometheus registry.
func NewSourceMetrics() SourceMetrics {
	return SourceMetrics{
		ListRepos: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_source_duration_seconds",
				Help: "Time spent sourcing repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_source_repos_total",
				Help: "Total number of sourced repositories",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_source_errors_total",
				Help: "Total number of sourcing errors",
			}, []string{}),
		},
	}
}

// ListRepos calls into the inner Source registers the observed results.
func (o *observedSource) ListRepos(ctx context.Context, results chan SourceResult) {
	var (
		err   error
		count float64
	)

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		o.metrics.ListRepos.Observe(secs, count, &err)
		logging.Log(o.log, "source.list-repos", &err)
	}(time.Now())

	uncounted := make(chan SourceResult)
	go func() {
		o.Source.ListRepos(ctx, uncounted)
		close(uncounted)
	}()

	var errs *multierror.Error
	for res := range uncounted {
		results <- res
		if res.Err != nil {
			errs = multierror.Append(errs, res.Err)
		}
		count++
	}
	if errs != nil {
		err = errs.ErrorOrNil()
	}
}

// NewObservedStore wraps the given Store with error logging,
// Prometheus metrics and tracing.
func NewObservedStore(
	s Store,
	l logging.ErrorLogger,
	m StoreMetrics,
	t trace.Tracer,
) *ObservedStore {
	return &ObservedStore{
		store:   s,
		log:     l,
		metrics: m,
		tracer:  t,
	}
}

// An ObservedStore wraps another Store with error logging,
// Prometheus metrics and tracing.
type ObservedStore struct {
	store   Store
	log     logging.ErrorLogger
	metrics StoreMetrics
	tracer  trace.Tracer
	txtrace *trace.Trace
	txctx   context.Context
}

// StoreMetrics encapsulates the Prometheus metrics of a Store.
type StoreMetrics struct {
	Transact               *metrics.OperationMetrics
	Done                   *metrics.OperationMetrics
	InsertRepos            *metrics.OperationMetrics
	DeleteRepos            *metrics.OperationMetrics
	UpsertRepos            *metrics.OperationMetrics
	UpsertSources          *metrics.OperationMetrics
	ListRepos              *metrics.OperationMetrics
	ListExternalRepoSpecs  *metrics.OperationMetrics
	UpsertExternalServices *metrics.OperationMetrics
	ListExternalServices   *metrics.OperationMetrics
	SetClonedRepos         *metrics.OperationMetrics
	CountNotClonedRepos    *metrics.OperationMetrics
	CountUserAddedRepos    *metrics.OperationMetrics
	EnqueueSyncJobs        *metrics.OperationMetrics
}

// MustRegister registers all metrics in StoreMetrics in the given
// prometheus.Registerer. It panics in case of failure.
func (sm StoreMetrics) MustRegister(r prometheus.Registerer) {
	for _, om := range []*metrics.OperationMetrics{
		sm.Transact,
		sm.Done,
		sm.ListRepos,
		sm.ListExternalRepoSpecs,
		sm.InsertRepos,
		sm.DeleteRepos,
		sm.UpsertRepos,
		sm.UpsertSources,
		sm.ListExternalServices,
		sm.UpsertExternalServices,
		sm.SetClonedRepos,
	} {
		r.MustRegister(om.Count)
		r.MustRegister(om.Duration)
		r.MustRegister(om.Errors)
	}
}

// NewStoreMetrics returns StoreMetrics that need to be registered
// in a Prometheus registry.
func NewStoreMetrics() StoreMetrics {
	return StoreMetrics{
		Transact: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_transact_duration_seconds",
				Help: "Time spent opening a transaction",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_transact_total",
				Help: "Total number of opened transactions",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_transact_errors_total",
				Help: "Total number of errors when opening a transaction",
			}, []string{}),
		},
		Done: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_done_duration_seconds",
				Help: "Time spent closing a transaction",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_done_total",
				Help: "Total number of closed transactions",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_done_errors_total",
				Help: "Total number of errors when closing a transaction",
			}, []string{}),
		},
		InsertRepos: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_insert_repos_duration_seconds",
				Help: "Time spent inserting repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_insert_repos_total",
				Help: "Total number of inserting repositories",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_insert_repos_errors_total",
				Help: "Total number of errors when inserting repos",
			}, []string{}),
		},
		DeleteRepos: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_delete_repos_duration_seconds",
				Help: "Time spent deleting repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_delete_repos_total",
				Help: "Total number of deleting repositories",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_delete_repos_errors_total",
				Help: "Total number of errors when deleting repos",
			}, []string{}),
		},
		UpsertRepos: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_upsert_repos_duration_seconds",
				Help: "Time spent upserting repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_upsert_repos_total",
				Help: "Total number of upserted repositories",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_upsert_repos_errors_total",
				Help: "Total number of errors when upserting repos",
			}, []string{}),
		},
		UpsertSources: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_upsert_sources_duration_seconds",
				Help: "Time spent upserting sources",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_upsert_sources_total",
				Help: "Total number of upserted sources",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_upsert_sources_errors_total",
				Help: "Total number of errors when upserting sources",
			}, []string{}),
		},
		ListRepos: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_list_repos_duration_seconds",
				Help: "Time spent listing repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_list_repos_total",
				Help: "Total number of listed repositories",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_list_repos_errors_total",
				Help: "Total number of errors when listing repos",
			}, []string{}),
		},
		ListExternalRepoSpecs: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_list_external_repo_specs_duration_seconds",
				Help: "Time spent listing external repo specs",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_list_external_repo_specs_total",
				Help: "Total number of listed external repo specs",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_list_external_repo_specs_errors_total",
				Help: "Total number of errors when listing external repo specs",
			}, []string{}),
		},
		UpsertExternalServices: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_external_serviceupdater_store_upsert_external_services_duration_seconds",
				Help: "Time spent upserting external_services",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_external_serviceupdater_store_upsert_external_services_total",
				Help: "Total number of upserted external_servicesitories",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_external_serviceupdater_store_upsert_external_services_errors_total",
				Help: "Total number of errors when upserting external_services",
			}, []string{}),
		},
		ListExternalServices: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_list_external_services_duration_seconds",
				Help: "Time spent listing external_services",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_list_external_services_total",
				Help: "Total number of listed external_servicesitories",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_list_external_services_errors_total",
				Help: "Total number of errors when listing external_services",
			}, []string{}),
		},
		SetClonedRepos: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_set_cloned_repos_duration_seconds",
				Help: "Time spent setting cloned repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_set_cloned_repos_total",
				Help: "Total number of set cloned repos calls",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_set_cloned_repos_errors_total",
				Help: "Total number of errors when setting cloned repos",
			}, []string{}),
		},
		CountNotClonedRepos: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_count_not_cloned_repos_duration_seconds",
				Help: "Time spent counting not-cloned repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_count_not_cloned_repos_total",
				Help: "Total number of count not-cloned repos calls",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_count_not_cloned_repos_errors_total",
				Help: "Total number of errors when counting not-cloned repos",
			}, []string{}),
		},
		CountUserAddedRepos: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_count_user_added_repos",
				Help: "Time spent counting the number of user added repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_count_user_added_repos_total",
				Help: "Total number of count user added repo calls",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_count_user_added_repos_errors_total",
				Help: "Total number of errors when counting user added repos",
			}, []string{}),
		},
	}
}

// Transact calls into the inner Store Transact method and
// returns an observed TxStore.
func (o *ObservedStore) Transact(ctx context.Context) (s TxStore, err error) {
	tr, ctx := o.trace(ctx, "Store.Transact")

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		o.metrics.Transact.Observe(secs, 1, &err)
		logging.Log(o.log, "store.transact", &err)
		if err != nil {
			tr.SetError(err)
			// Finish is called in Done in the non-error case
			tr.Finish()
		}
	}(time.Now())

	s, err = o.store.(Transactor).Transact(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "observed store")
	}

	return &ObservedStore{
		store:   s,
		log:     o.log,
		metrics: o.metrics,
		tracer:  o.tracer,
		txtrace: tr,
		txctx:   ctx,
	}, nil
}

// With calls the With method of the underlying store if it exists,
// otherwise it returns the store unchanged.
// It implements the WithStore interface.
func (o *ObservedStore) With(db dbutil.DB) Store {
	if ws, ok := o.store.(WithStore); ok {
		return ws.With(db)
	}

	return o.store
}

// Done calls into the inner Store Done method.
func (o *ObservedStore) Done(err error) error {
	tr := o.txtrace
	tr.LogFields(otlog.String("event", "Store.Done"))

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		done := false

		if err != nil {
			done = true
			tr.SetError(err)
			o.metrics.Done.Observe(secs, 1, &err)
			logging.Log(o.log, "store.done", &err)
		}

		if !done {
			o.metrics.Done.Observe(secs, 1, nil)
		}

		tr.Finish()
	}(time.Now())

	return o.store.(TxStore).Done(err)
}

// ListExternalServices calls into the inner Store and registers the observed results.
func (o *ObservedStore) ListExternalServices(ctx context.Context, args StoreListExternalServicesArgs) (es []*ExternalService, err error) {
	tr, ctx := o.trace(ctx, "Store.ListExternalServices")
	tr.LogFields(
		otlog.Object("args.ids", args.IDs),
		otlog.Object("args.kinds", args.Kinds),
	)

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		count := float64(len(es))

		o.metrics.ListExternalServices.Observe(secs, count, &err)
		logging.Log(o.log, "store.list-external-services", &err,
			"args", fmt.Sprintf("%+v", args),
			"count", len(es),
		)

		tr.LogFields(
			otlog.Int("count", len(es)),
			otlog.Object("names", ExternalServices(es).DisplayNames()),
			otlog.Object("urns", ExternalServices(es).URNs()),
		)
		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.ListExternalServices(ctx, args)
}

// UpsertExternalServices calls into the inner Store and registers the observed results.
func (o *ObservedStore) UpsertExternalServices(ctx context.Context, svcs ...*ExternalService) (err error) {
	tr, ctx := o.trace(ctx, "Store.UpsertExternalServices")
	tr.LogFields(
		otlog.Int("count", len(svcs)),
		otlog.Object("names", ExternalServices(svcs).DisplayNames()),
		otlog.Object("urns", ExternalServices(svcs).URNs()),
	)

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		count := float64(len(svcs))

		o.metrics.UpsertExternalServices.Observe(secs, count, &err)
		logging.Log(o.log, "store.upsert-external-services", &err,
			"count", len(svcs),
			"names", ExternalServices(svcs).DisplayNames(),
		)

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.UpsertExternalServices(ctx, svcs...)
}

// InsertRepos calls into the inner Store and registers the observed results.
func (o *ObservedStore) InsertRepos(ctx context.Context, repos ...*Repo) (err error) {
	tr, ctx := o.trace(ctx, "Store.InsertRepos")
	tr.LogFields(otlog.Int("count", len(repos)))

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		count := float64(len(repos))

		o.metrics.InsertRepos.Observe(secs, count, &err)
		logging.Log(o.log, "store.insert-repos", &err, "count", len(repos))

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.InsertRepos(ctx, repos...)
}

// DeleteRepos calls into the inner Store and registers the observed results.
func (o *ObservedStore) DeleteRepos(ctx context.Context, ids ...api.RepoID) (err error) {
	tr, ctx := o.trace(ctx, "Store.DeleteRepos")
	tr.LogFields(otlog.Int("count", len(ids)))

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		count := float64(len(ids))

		o.metrics.InsertRepos.Observe(secs, count, &err)
		logging.Log(o.log, "store.delete-repos", &err, "count", len(ids))

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.DeleteRepos(ctx, ids...)
}

// ListRepos calls into the inner Store and registers the observed results.
func (o *ObservedStore) ListRepos(ctx context.Context, args StoreListReposArgs) (rs []*Repo, err error) {
	tr, ctx := o.trace(ctx, "Store.ListRepos")
	tr.LogFields(
		otlog.Object("args.names", args.Names),
		otlog.Object("args.ids", args.IDs),
		otlog.Object("args.kinds", args.Kinds),
	)

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		count := float64(len(rs))

		o.metrics.ListRepos.Observe(secs, count, &err)
		logging.Log(o.log, "store.list-repos", &err,
			"args", fmt.Sprintf("%+v", args),
			"count", len(rs),
		)

		tr.LogFields(otlog.Int("count", len(rs)))
		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.ListRepos(ctx, args)
}

// ListExternalRepoSpecs calls into the inner Store and registers the observed results.
func (o *ObservedStore) ListExternalRepoSpecs(ctx context.Context) (ids map[api.ExternalRepoSpec]struct{}, err error) {
	tr, ctx := o.trace(ctx, "Store.ListExternalRepoSpecs")

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		count := float64(len(ids))

		o.metrics.ListExternalRepoSpecs.Observe(secs, count, &err)
		logging.Log(o.log, "store.list-external-repo-specs", &err,
			"count", len(ids),
		)

		tr.LogFields(otlog.Int("count", len(ids)))
		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.ListExternalRepoSpecs(ctx)
}

// UpsertRepos calls into the inner Store and registers the observed results.
func (o *ObservedStore) UpsertRepos(ctx context.Context, repos ...*Repo) (err error) {
	tr, ctx := o.trace(ctx, "Store.UpsertRepos")
	tr.LogFields(otlog.Int("count", len(repos)))

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		count := float64(len(repos))

		o.metrics.UpsertRepos.Observe(secs, count, &err)
		logging.Log(o.log, "store.upsert-repos", &err, "count", len(repos))

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.UpsertRepos(ctx, repos...)
}

// UpsertSources calls into the inner Store and registers the observed results.
func (o *ObservedStore) UpsertSources(ctx context.Context, added, modified, deleted map[api.RepoID][]SourceInfo) (err error) {
	tr, ctx := o.trace(ctx, "Store.UpsertSources")
	tr.LogFields(otlog.Int("count", len(added)+len(deleted)))

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		count := float64(len(added) + len(modified) + len(deleted))

		o.metrics.UpsertSources.Observe(secs, count, &err)
		logging.Log(o.log, "store.upsert-sources", &err, "count", count)

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.UpsertSources(ctx, added, modified, deleted)
}

// SetClonedRepos calls into the inner Store and registers the observed results.
func (o *ObservedStore) SetClonedRepos(ctx context.Context, repoNames ...string) (err error) {
	tr, ctx := o.trace(ctx, "Store.SetClonedRepos")
	tr.LogFields(otlog.Int("count", len(repoNames)))

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		count := float64(len(repoNames))

		o.metrics.SetClonedRepos.Observe(secs, count, &err)
		logging.Log(o.log, "store.set-cloned-repos", &err, "count", len(repoNames))

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.SetClonedRepos(ctx, repoNames...)
}

// CountNotClonedRepos calls into the inner Store and registers the observed results.
func (o *ObservedStore) CountNotClonedRepos(ctx context.Context) (count uint64, err error) {
	tr, ctx := o.trace(ctx, "Store.CountNotClonedRepos")

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()

		o.metrics.CountNotClonedRepos.Observe(secs, float64(count), &err)
		logging.Log(o.log, "store.count-not-cloned-repos", &err, "count", count)

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.CountNotClonedRepos(ctx)
}

// CountUserAddedRepos calls into the inner Store and registers the observed results.
func (o *ObservedStore) CountUserAddedRepos(ctx context.Context) (count uint64, err error) {
	tr, ctx := o.trace(ctx, "Store.CountUserAddedRepos")

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()

		o.metrics.CountUserAddedRepos.Observe(secs, float64(count), &err)
		logging.Log(o.log, "store.count-user-added-repos", &err, "count", count)

		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.CountUserAddedRepos(ctx)
}

func (o *ObservedStore) EnqueueSyncJobs(ctx context.Context, ignoreSiteAdmin bool) (err error) {
	tr, ctx := o.trace(ctx, "Store.EnqueueSyncJobs")

	defer func(began time.Time) {
		secs := time.Since(began).Seconds()
		o.metrics.EnqueueSyncJobs.Observe(secs, 0, &err)
		tr.SetError(err)
		tr.Finish()
	}(time.Now())

	return o.store.EnqueueSyncJobs(ctx, ignoreSiteAdmin)
}

func (o *ObservedStore) trace(ctx context.Context, family string) (*trace.Trace, context.Context) {
	txctx := o.txctx
	if txctx == nil {
		txctx = ctx
	}
	tr, txctx := o.tracer.New(txctx, family, "")
	ctx = trace.CopyContext(ctx, txctx)
	return tr, ctx
}
