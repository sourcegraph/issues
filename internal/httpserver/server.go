package httpserver

import (
	"context"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/trace/ot"
)

type server struct {
	server *http.Server
	once   sync.Once
}

// New returns a BackgroundRoutine that maintains an HTTP server listening on the given
// port with a router configured with the given function. All servers will respond 200
// to requests to /healthz.
func New(port int, setupRoutes func(router *mux.Router)) goroutine.BackgroundRoutine {
	host := ""
	if env.InsecureDev {
		host = "127.0.0.1"
	}

	router := mux.NewRouter()
	setupRoutes(router)
	router.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return &server{
		server: &http.Server{
			Addr:    net.JoinHostPort(host, strconv.FormatInt(int64(port), 10)),
			Handler: ot.Middleware(router),
		},
	}
}

func (s *server) Start() {
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		log15.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func (s *server) Stop() {
	s.once.Do(func() {
		if err := s.server.Shutdown(context.Background()); err != nil {
			log15.Error("Failed to shutdown server", "error", err)
		}
	})
}
