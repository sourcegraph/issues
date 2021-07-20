package repos

import (
	"context"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/internal/logging"
	"github.com/sourcegraph/sourcegraph/internal/metrics"
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

// StoreMetrics encapsulates the Prometheus metrics of a Store.
type StoreMetrics struct {
	Transact                        *metrics.OperationMetrics
	Done                            *metrics.OperationMetrics
	CreateExternalServiceRepo       *metrics.OperationMetrics
	UpdateExternalServiceRepo       *metrics.OperationMetrics
	DeleteExternalServiceRepo       *metrics.OperationMetrics
	DeleteExternalServiceReposNotIn *metrics.OperationMetrics
	UpsertRepos                     *metrics.OperationMetrics
	UpsertSources                   *metrics.OperationMetrics
	ListExternalRepoSpecs           *metrics.OperationMetrics
	GetExternalService              *metrics.OperationMetrics
	SetClonedRepos                  *metrics.OperationMetrics
	CountNotClonedRepos             *metrics.OperationMetrics
	CountUserAddedRepos             *metrics.OperationMetrics
	EnqueueSyncJobs                 *metrics.OperationMetrics
}

// MustRegister registers all metrics in StoreMetrics in the given
// prometheus.Registerer. It panics in case of failure.
func (sm StoreMetrics) MustRegister(r prometheus.Registerer) {
	for _, om := range []*metrics.OperationMetrics{
		sm.Transact,
		sm.Done,
		sm.ListExternalRepoSpecs,
		sm.CreateExternalServiceRepo,
		sm.UpdateExternalServiceRepo,
		sm.DeleteExternalServiceRepo,
		sm.DeleteExternalServiceReposNotIn,
		sm.UpsertRepos,
		sm.UpsertSources,
		sm.GetExternalService,
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
		CreateExternalServiceRepo: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repocreater_store_create_external_service_repo_duration_seconds",
				Help: "Time spent creating external service repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repocreater_store_create_external_service_repo_total",
				Help: "Total number of created external service repos",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repocreater_store_create_external_service_repo_errors_total",
				Help: "Total number of errors when creating external service repos",
			}, []string{}),
		},
		UpdateExternalServiceRepo: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_update_external_service_repo_duration_seconds",
				Help: "Time spent updating external service repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_update_external_service_repo_total",
				Help: "Total number of updated external service repos",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_update_external_service_repo_errors_total",
				Help: "Total number of errors when updating external service repos",
			}, []string{}),
		},
		DeleteExternalServiceRepo: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_delete_external_service_repo_duration_seconds",
				Help: "Time spent deleting external service repos",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_delete_external_service_repo_total",
				Help: "Total number of external service repo deletions",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_delete_external_service_repo_errors_total",
				Help: "Total number of errors when deleting external service repos",
			}, []string{}),
		},
		DeleteExternalServiceReposNotIn: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_repoupdater_store_delete_exernal_service_repos_not_in_duration_seconds",
				Help: "Time spent calling DeleteExternalServiceReposNotIn",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_delete_exernal_service_repos_not_in_total",
				Help: "Total number of calls to DeleteExternalServiceReposNotIn",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_repoupdater_store_delete_exernal_service_repos_not_in_errors_total",
				Help: "Total number of errors when calling DeleteExternalServiceReposNotIn",
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
		GetExternalService: &metrics.OperationMetrics{
			Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name: "src_external_serviceupdater_store_get_external_service_duration_seconds",
				Help: "Time spent getting external_services",
			}, []string{}),
			Count: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_external_serviceupdater_store_get_external_service_total",
				Help: "Total number of get external_service calls",
			}, []string{}),
			Errors: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "src_external_serviceupdater_store_get_external_service_errors_total",
				Help: "Total number of errors when getting external_services",
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
