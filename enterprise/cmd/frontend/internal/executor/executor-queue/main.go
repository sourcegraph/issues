package main

import (
	"context"
	"log"

	"github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/config"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/queues/batches"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/queues/codeintel"
	apiserver "github.com/sourcegraph/sourcegraph/enterprise/cmd/executor-queue/internal/server"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
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

	for _, config := range configs {
		if err := config.Validate(); err != nil {
			log.Fatalf("failed to load config: %s", err)
		}
	}

	// Initialize queues
	queueOptions := map[string]apiserver.QueueOptions{
		"codeintel": codeintel.QueueOptions(db, codeintelConfig, observationContext),
		"batches":   batches.QueueOptions(db, batchesConfig, observationContext),
	}

	for queueName, options := range queueOptions {
		prometheus.DefaultRegisterer.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name:        "src_executor_queue_total",
			Help:        "Total number of jobs in the queued state.",
			ConstLabels: map[string]string{"queue": queueName},
		}, func() float64 {
			// TODO(efritz) - do not count soft-deleted code intel index records
			count, err := options.Store.QueuedCount(context.Background(), nil)
			if err != nil {
				log15.Error("Failed to get queued job count", "queue", queueName, "error", err)
			}

			return float64(count)
		}))
	}

	server := apiserver.NewServer(serviceConfig.ServerOptions(queueOptions), observationContext)
	goroutine.MonitorBackgroundRoutines(context.Background(), server)
}
