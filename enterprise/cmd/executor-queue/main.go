package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/config"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/queues/batches"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/queues/codeintel"
	apiserver "github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/server"
	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/database/dbconn"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/logging"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/tracer"
)

type configuration interface {
	Load()
	Validate() error
}

func main() {
	serviceConfig := &Config{}
	sharedConfig := &config.SharedConfig{}
	codeintelConfig := &codeintel.Config{Shared: sharedConfig}
	batchesConfig := &batches.Config{Shared: sharedConfig}
	configs := []configuration{serviceConfig, sharedConfig, codeintelConfig, batchesConfig}

	for _, config := range configs {
		config.Load()
	}

	env.Lock()
	env.HandleHelpFlag()

	logging.Init()
	tracer.Init()
	trace.Init(true)

	for _, config := range configs {
		if err := config.Validate(); err != nil {
			log.Fatalf("failed to load config: %s", err)
		}
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

	// Connect to databases
	db := connectToDatabase()

	// Migrations may take a while, but after they're done we'll immediately
	// spin up a server and can accept traffic. Inform external clients we'll
	// be ready for traffic.
	close(ready)

	// Initialize queues
	queueOptions := map[string]apiserver.QueueOptions{
		"codeintel": codeintel.QueueOptions(db, codeintelConfig, observationContext),
		"batches":   batches.QueueOptions(db, batchesConfig, observationContext),
	}

	for queueName, options := range queueOptions {
		// Make local copy of queue name for capture below
		queueName, store := queueName, options.Store

		prometheus.DefaultRegisterer.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name:        "src_executor_total",
			Help:        "Total number of jobs in the queued state.",
			ConstLabels: map[string]string{"queue": queueName},
		}, func() float64 {
			// TODO(efritz) - do not count soft-deleted code intel index records
			count, err := store.QueuedCount(context.Background(), nil)
			if err != nil {
				log15.Error("Failed to get queued job count", "queue", queueName, "error", err)
			}

			return float64(count)
		}))
	}

	server := apiserver.NewServer(serviceConfig.ServerOptions(queueOptions), observationContext)
	goroutine.MonitorBackgroundRoutines(context.Background(), server)
}

func connectToDatabase() *sql.DB {
	postgresDSN := conf.Get().ServiceConnections.PostgresDSN
	conf.Watch(func() {
		if newDSN := conf.Get().ServiceConnections.PostgresDSN; postgresDSN != newDSN {
			log.Fatalf("detected database DSN change, restarting to take effect: %s", newDSN)
		}
	})

	db, err := dbconn.New(dbconn.Opts{DSN: postgresDSN, DBName: "frontend", AppName: "executor-queue"})
	if err != nil {
		log.Fatalf("failed to initialize store: %s", err)
	}

	return db
}
