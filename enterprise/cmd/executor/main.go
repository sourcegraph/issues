package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sourcegraph/sourcegraph/enterprise/cmd/executor/internal/worker"
	"github.com/sourcegraph/sourcegraph/internal/debugserver"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/httpserver"
	"github.com/sourcegraph/sourcegraph/internal/logging"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/internal/workerutil"
)

func main() {
	config := &Config{}
	config.Load()

	env.Lock()
	env.HandleHelpFlag()

	logging.Init()
	trace.Init(false)

	if err := config.Validate(); err != nil {
		log.Fatalf("failed to read config: %s", err)
	}

	// Initialize tracing/metrics
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	// Ready immediately
	ready := make(chan struct{})
	close(ready)
	go debugserver.NewServerRoutine(ready).Start()

	routines := []goroutine.BackgroundRoutine{
		worker.NewWorker(config.APIWorkerOptions(nil), observationContext),
	}
	if !config.DisableHealthServer {
		routines = append(routines, httpserver.NewFromAddr(fmt.Sprintf(":%d", config.HealthServerPort), &http.Server{
			ReadTimeout:  75 * time.Second,
			WriteTimeout: 10 * time.Minute,
			Handler:      httpserver.NewHandler(nil),
		}))
	}

	goroutine.MonitorBackgroundRoutines(context.Background(), routines...)
}

func makeWorkerMetrics(queueName string) workerutil.WorkerMetrics {
	observationContext := &observation.Context{
		Logger:     log15.Root(),
		Tracer:     &trace.Tracer{Tracer: opentracing.GlobalTracer()},
		Registerer: prometheus.DefaultRegisterer,
	}

	return workerutil.NewMetrics(observationContext, "executor_processor", map[string]string{
		"queue": queueName,
	})
}
