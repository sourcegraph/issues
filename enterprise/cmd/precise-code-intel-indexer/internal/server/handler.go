package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/inconshreveable/log15"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/queue/types"
)

func (s *Server) setupRoutes(router *mux.Router) {
	router.Path("/dequeue").Methods("POST").HandlerFunc(s.handleDequeue)
	router.Path("/setlog").Methods("POST").HandlerFunc(s.handleSetLogContents)
	router.Path("/complete").Methods("POST").HandlerFunc(s.handleComplete)
	router.Path("/heartbeat").Methods("POST").HandlerFunc(s.handleHeartbeat)
}

// POST /dequeue
func (s *Server) handleDequeue(w http.ResponseWriter, r *http.Request) {
	var payload types.DequeueRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	index, dequeued, err := s.indexManager.Dequeue(r.Context(), payload.IndexerName)
	if err != nil {
		log15.Error("Failed to dequeue index", "err", err)
		http.Error(w, fmt.Sprintf("failed to dequeue index: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	if !dequeued {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, index)
}

// POST /setlog
func (s *Server) handleSetLogContents(w http.ResponseWriter, r *http.Request) {
	var payload types.SetLogRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	if err := s.indexManager.SetLogContents(r.Context(), payload.IndexerName, payload.IndexID, payload.Contents); err != nil {
		log15.Error("Failed to set log contents", "err", err)
		http.Error(w, fmt.Sprintf("failed to set log contents: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /complete
func (s *Server) handleComplete(w http.ResponseWriter, r *http.Request) {
	var payload types.CompleteRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	found, err := s.indexManager.Complete(r.Context(), payload.IndexerName, payload.IndexID, payload.ErrorMessage)
	if err != nil {
		log15.Error("Failed to complete index job", "err", err)
		http.Error(w, fmt.Sprintf("failed to complete index job: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /heartbeat
func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var payload types.HeartbeatRequest
	if !decodeBody(w, r, &payload) {
		return
	}

	if err := s.indexManager.Heartbeat(r.Context(), payload.IndexerName, payload.IndexIDs); err != nil {
		log15.Error("Failed to acknowledge heartbeat", "err", err)
		http.Error(w, fmt.Sprintf("failed to acknowledge heartbeat: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
