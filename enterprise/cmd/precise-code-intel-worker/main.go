package main

import (
	"database/sql"
	"log"

	"github.com/gorilla/mux"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	commitupdater "github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/commit-updater"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/metrics"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/resetter"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/worker"
	bundles "github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/bundles/client"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/commits"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/store"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/db/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/httpserver"
	"github.com/sourcegraph/sourcegraph/internal/logging"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/sqliteutil"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/tracer"
)

const Port = 3188

func main() {
	env.Lock()
	env.HandleHelpFlag()
	logging.Init()
	tracer.Init()
	trace.Init(true)

	sqliteutil.MustRegisterSqlite3WithPcre()

	var (
		bundleManagerURL      = mustGet(rawBundleManagerURL, "PRECISE_CODE_INTEL_BUNDLE_MANAGER_URL")
		workerPollInterval    = mustParseInterval(rawWorkerPollInterval, "PRECISE_CODE_INTEL_WORKER_POLL_INTERVAL")
		workerConcurrency     = mustParseInt(rawWorkerConcurrency, "PRECISE_CODE_INTEL_WORKER_CONCURRENCY")
		workerBudget          = mustParseInt64(rawWorkerBudget, "PRECISE_CODE_INTEL_WORKER_BUDGET")
		resetInterval         = mustParseInterval(rawResetInterval, "PRECISE_CODE_INTEL_RESET_INTERVAL")
		commitUpdaterInterval = mustParseInterval(rawCommitUpdaterInterval, "PRECISE_CODE_INTEL_COMMIT_UPDATER_INTERVAL")
	)

	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	codeIntelDB := mustInitializeCodeIntelDatabase()

	store := store.NewObserved(mustInitializeStore(), observationContext)
	MustRegisterQueueMonitor(observationContext.Registerer, store)
	workerMetrics := metrics.NewWorkerMetrics(observationContext)
	resetterMetrics := resetter.NewResetterMetrics(prometheus.DefaultRegisterer)
	server := httpserver.New(Port, func(router *mux.Router) {})
	uploadResetter := resetter.NewUploadResetter(store, resetInterval, resetterMetrics)
	commitUpdater := commitupdater.NewUpdater(
		store,
		commits.NewUpdater(store, gitserver.DefaultClient),
		commitupdater.UpdaterOptions{
			Interval: commitUpdaterInterval,
		},
	)
	worker := worker.NewWorker(
		store,
		codeIntelDB,
		bundles.New(codeIntelDB, observationContext, bundleManagerURL),
		gitserver.DefaultClient,
		workerPollInterval,
		workerConcurrency,
		workerBudget,
		workerMetrics,
		observationContext,
	)

	go debugserver.Start()
	goroutine.MonitorBackgroundRoutines(server, uploadResetter, commitUpdater, worker)
}

func mustInitializeStore() store.Store {
	postgresDSN := conf.Get().ServiceConnections.PostgresDSN
	conf.Watch(func() {
		if newDSN := conf.Get().ServiceConnections.PostgresDSN; postgresDSN != newDSN {
			log.Fatalf("detected database DSN change, restarting to take effect: %s", newDSN)
		}
	})

	if err := dbconn.SetupGlobalConnection(postgresDSN); err != nil {
		log.Fatalf("failed to connect to frontend database: %s", err)
	}

	return store.NewWithDB(dbconn.Global)
}

func mustInitializeCodeIntelDatabase() *sql.DB {
	postgresDSN := conf.Get().ServiceConnections.CodeIntelPostgresDSN
	conf.Watch(func() {
		if newDSN := conf.Get().ServiceConnections.CodeIntelPostgresDSN; postgresDSN != newDSN {
			log.Fatalf("detected database DSN change, restarting to take effect: %s", newDSN)
		}
	})

	db, err := dbconn.New(postgresDSN, "_codeintel")
	if err != nil {
		log.Fatalf("failed to connect to codeintel database: %s", err)
	}

	if err := dbconn.MigrateDB(db, "codeintel"); err != nil {
		log.Fatalf("failed to perform codeintel database migration: %s", err)
	}

	return db
}
