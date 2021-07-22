package server

import (
	"fmt"
	"net/http"

	"github.com/derision-test/glock"
	"github.com/gorilla/mux"

	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/httpserver"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

// NewServer returns an HTTP job queue server.
func NewServer(options Options, queueOptions []QueueOptions, observationContext *observation.Context) goroutine.BackgroundRoutine {
	addr := fmt.Sprintf(":%d", options.Port)

	queueMetrics := newQueueMetrics(observationContext)
	serverHandler := ot.Middleware(httpserver.NewHandler(func(router *mux.Router) {
		for _, queueOption := range queueOptions {
			handler := newHandlerWithMetrics(options, queueOption, glock.NewRealClock(), queueMetrics)
			subRouter := router.PathPrefix(fmt.Sprintf("/%s/", queueOption.Name)).Subrouter()
			handler.setupRoutes(subRouter)
		}
	}))

	server := httpserver.NewFromAddr(addr, &http.Server{Handler: serverHandler})

	return server
}
