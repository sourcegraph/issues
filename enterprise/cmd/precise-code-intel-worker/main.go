package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/precise-code-intel-worker/internal/worker"
	eiauthz "github.com/sourcegraph/sourcegraph/enterprise/internal/authz"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/gitserver"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/dbstore"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/lsifstore"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/stores/uploadstore"
	"github.com/sourcegraph/sourcegraph/internal/authz"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/encryption/keyring"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/httpserver"
	"github.com/sourcegraph/sourcegraph/internal/logging"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/tracer"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
	dbworkerstore "github.com/sourcegraph/sourcegraph/internal/workerutil/dbworker/store"
)

const addr = ":3188"

func main() {
	config := &Config{}
	config.Load()

	env.Lock()
	env.HandleHelpFlag()
	logging.Init()
	tracer.Init()
	trace.Init(true)

	if err := config.Validate(); err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}

	// Initialize tracing/metrics
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	// Start debug server
	ready := make(chan struct{})
	go debugserver.NewServerRoutine(ready).Start()

	if err := keyring.Init(context.Background()); err != nil {
		log.Fatalf("Failed to intialise keyring: %v", err)
	}

	// Connect to databases
	db := mustInitializeDB()
	codeIntelDB := mustInitializeCodeIntelDB()

	// Migrations may take a while, but after they're done we'll immediately
	// spin up a server and can accept traffic. Inform external clients we'll
	// be ready for traffic.
	close(ready)

	// Initialize stores
	dbStore := dbstore.NewWithDB(db, observationContext)
	workerStore := dbstore.WorkerutilUploadStore(dbStore, observationContext)
	lsifStore := lsifstore.NewStore(codeIntelDB, observationContext)
	gitserverClient := gitserver.New(dbStore, observationContext)

	uploadStore, err := uploadstore.CreateLazy(context.Background(), config.UploadStoreConfig, observationContext)
	if err != nil {
		log.Fatalf("Failed to create upload store: %s", err)
	}
	if err := initializeUploadStore(context.Background(), uploadStore); err != nil {
		log.Fatalf("Failed to initialize upload store: %s", err)
	}

	// Initialize metrics
	mustRegisterQueueMetric(observationContext, workerStore)

	// Initialize worker
	worker := worker.NewWorker(
		&worker.DBStoreShim{Store: dbStore},
		workerStore,
		&worker.LSIFStoreShim{Store: lsifStore},
		uploadStore,
		gitserverClient,
		config.WorkerPollInterval,
		config.WorkerConcurrency,
		config.WorkerBudget,
		makeWorkerMetrics(observationContext),
	)

	// Initialize health server
	server := httpserver.NewFromAddr(addr, &http.Server{
		ReadTimeout:  75 * time.Second,
		WriteTimeout: 10 * time.Minute,
		Handler:      httpserver.NewHandler(nil),
	})

	// Go!
	goroutine.MonitorBackgroundRoutines(context.Background(), worker, server)
}

func mustInitializeDB() *sql.DB {
	postgresDSN := conf.Get().ServiceConnections.PostgresDSN
	conf.Watch(func() {
		if newDSN := conf.Get().ServiceConnections.PostgresDSN; postgresDSN != newDSN {
			log.Fatalf("Detected database DSN change, restarting to take effect: %s", newDSN)
		}
	})

	opts := dbconn.Opts{DSN: postgresDSN, DBName: "frontend", AppName: "precise-code-intel-worker"}
	if err := dbconn.SetupGlobalConnection(opts); err != nil {
		log.Fatalf("Failed to connect to frontend database: %s", err)
	}

	//
	// START FLAILING

	ctx := context.Background()
	go func() {
		for range time.NewTicker(5 * time.Second).C {
			allowAccessByDefault, authzProviders, _, _ := eiauthz.ProvidersFromConfig(ctx, conf.Get(), database.ExternalServices(dbconn.Global))
			authz.SetProviders(allowAccessByDefault, authzProviders)
		}
	}()

	// END FLAILING
	//

	return dbconn.Global
}

func mustInitializeCodeIntelDB() *sql.DB {
	postgresDSN := conf.Get().ServiceConnections.CodeIntelPostgresDSN
	conf.Watch(func() {
		if newDSN := conf.Get().ServiceConnections.CodeIntelPostgresDSN; postgresDSN != newDSN {
			log.Fatalf("Detected codeintel database DSN change, restarting to take effect: %s", newDSN)
		}
	})

	db, err := dbconn.New(dbconn.Opts{DSN: postgresDSN, DBName: "codeintel", AppName: "precise-code-intel-worker"})
	if err != nil {
		log.Fatalf("Failed to connect to codeintel database: %s", err)
	}

	if err := dbconn.MigrateDB(db, dbconn.CodeIntel); err != nil {
		log.Fatalf("Failed to perform codeintel database migration: %s", err)
	}

	return db
}

func mustRegisterQueueMetric(observationContext *observation.Context, workerStore dbworkerstore.Store) {
	observationContext.Registerer.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "src_codeintel_upload_total",
		Help: "Total number of uploads in the queued state.",
	}, func() float64 {
		count, err := workerStore.QueuedCount(context.Background(), nil)
		if err != nil {
			log15.Error("Failed to determine queue size", "err", err)
		}

		return float64(count)
	}))
}

func makeWorkerMetrics(observationContext *observation.Context) workerutil.WorkerMetrics {
	return workerutil.NewMetrics(observationContext, "codeintel_upload_processor", nil)
}

func initializeUploadStore(ctx context.Context, uploadStore uploadstore.Store) error {
	for {
		if err := uploadStore.Init(ctx); err == nil || !isRequestError(err) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func isRequestError(err error) bool {
	return errors.HasType(err, &smithyhttp.RequestSendError{})
}
