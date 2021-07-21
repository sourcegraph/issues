package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/derision-test/glock"
	"github.com/gorilla/mux"
	"github.com/inconshreveable/log15"

	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/httpserver"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

// NewServer returns an HTTP job queue server.
func NewServer(options Options, queueOptions []QueueOptions, observationContext *observation.Context) goroutine.BackgroundRoutine {
	addr := fmt.Sprintf(":%d", options.Port)

	routines := goroutine.CombinedRoutine{}
	queueMetrics := newQueueMetrics(observationContext)
	serverHandler := ot.Middleware(httpserver.NewHandler(func(router *mux.Router) {
		for _, queueOption := range queueOptions {
			handler := newHandlerWithMetrics(options, queueOption, glock.NewRealClock(), queueMetrics)
			subRouter := router.PathPrefix(fmt.Sprintf("/%s/", queueOption.Name)).Subrouter()
			handler.setupRoutes(subRouter)

			janitor := goroutine.NewPeriodicGoroutine(context.Background(), options.CleanupInterval, &handlerWrapper{handler})
			routines = append(routines, janitor)
		}
	}))

	server := httpserver.NewFromAddr(addr, &http.Server{Handler: serverHandler})
	routines = append(routines, server)

	return routines
}

type handlerWrapper struct{ handler *handler }

var _ goroutine.Handler = &handlerWrapper{}

func (hw *handlerWrapper) Handle(ctx context.Context) error { return hw.handler.cleanup(ctx) }
func (hw *handlerWrapper) HandleError(err error)            { log15.Error("Failed to requeue jobs", "err", err) }
