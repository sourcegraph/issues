package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/inconshreveable/log15"

	apiclient "github.com/sourcegraph/sourcegraph/enterprise/internal/executor"
)

func (h *handler) setupRoutes(router *mux.Router) {
	var names []string
	for queueName := range h.options.QueueOptions {
		names = append(names, regexp.QuoteMeta(queueName))
	}

	routes := map[string]func(w http.ResponseWriter, r *http.Request){
		"dequeue":              h.handleDequeue,
		"addExecutionLogEntry": h.handleAddExecutionLogEntry,
		"markComplete":         h.handleMarkComplete,
		"markErrored":          h.handleMarkErrored,
		"markFailed":           h.handleMarkFailed,
	}
	for path, handler := range routes {
		router.Path(fmt.Sprintf("/{queueName:(?:%s)}/%s", strings.Join(names, "|"), path)).Methods("POST").HandlerFunc(handler)
	}

	router.Path("/heartbeat").Methods("POST").HandlerFunc(h.handleHeartbeat)
}

// POST /{queueName}/dequeue
func (h *handler) handleDequeue(w http.ResponseWriter, r *http.Request) {
	var payload apiclient.DequeueRequest

	h.wrapHandler(w, r, &payload, func() (int, interface{}, error) {
		job, dequeued, err := h.dequeue(r.Context(), mux.Vars(r)["queueName"], payload.ExecutorName, payload.ExecutorHostname)
		if !dequeued {
			return http.StatusNoContent, nil, err
		}

		return http.StatusOK, job, err
	})
}

// POST /{queueName}/addExecutionLogEntry
func (h *handler) handleAddExecutionLogEntry(w http.ResponseWriter, r *http.Request) {
	var payload apiclient.AddExecutionLogEntryRequest

	h.wrapHandler(w, r, &payload, func() (int, interface{}, error) {
		err := h.addExecutionLogEntry(r.Context(), mux.Vars(r)["queueName"], payload.ExecutorName, payload.JobID, payload.ExecutionLogEntry)
		return http.StatusNoContent, nil, err
	})
}

// POST /{queueName}/markComplete
func (h *handler) handleMarkComplete(w http.ResponseWriter, r *http.Request) {
	var payload apiclient.MarkCompleteRequest

	h.wrapHandler(w, r, &payload, func() (int, interface{}, error) {
		err := h.markComplete(r.Context(), mux.Vars(r)["queueName"], payload.ExecutorName, payload.JobID)
		if err == ErrUnknownJob {
			return http.StatusNotFound, nil, nil
		}

		return http.StatusNoContent, nil, err
	})
}

// POST /{queueName}/markErrored
func (h *handler) handleMarkErrored(w http.ResponseWriter, r *http.Request) {
	var payload apiclient.MarkErroredRequest

	h.wrapHandler(w, r, &payload, func() (int, interface{}, error) {
		err := h.markErrored(r.Context(), mux.Vars(r)["queueName"], payload.ExecutorName, payload.JobID, payload.ErrorMessage)
		if err == ErrUnknownJob {
			return http.StatusNotFound, nil, nil
		}

		return http.StatusNoContent, nil, err
	})
}

// POST /{queueName}/markFailed
func (h *handler) handleMarkFailed(w http.ResponseWriter, r *http.Request) {
	var payload apiclient.MarkErroredRequest

	h.wrapHandler(w, r, &payload, func() (int, interface{}, error) {
		err := h.markFailed(r.Context(), mux.Vars(r)["queueName"], payload.ExecutorName, payload.JobID, payload.ErrorMessage)
		if err == ErrUnknownJob {
			return http.StatusNotFound, nil, nil
		}

		return http.StatusNoContent, nil, err
	})
}

// POST /heartbeat
func (h *handler) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var payload apiclient.HeartbeatRequest

	h.wrapHandler(w, r, &payload, func() (int, interface{}, error) {
		unknownIDs, err := h.heartbeat(r.Context(), payload.ExecutorName, payload.JobIDs)
		return http.StatusOK, unknownIDs, err
	})
}

type errorResponse struct {
	Error string `json:"error"`
}

// wrapHandler decodes the request body into the given payload pointer, then calls the given
// handler function. If the body cannot be decoded, a 400 BadRequest is returned and the handler
// function is not called. If the handler function returns an error, a 500 Internal Server Error
// is returned. Otherwise, the response status will match the status code value returned from the
// handler, and the payload value returned from the handler is encoded and written to the
// response body.
func (h *handler) wrapHandler(w http.ResponseWriter, r *http.Request, payload interface{}, handler func() (int, interface{}, error)) {
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal payload: %s", err.Error()), http.StatusBadRequest)
		return
	}

	status, payload, err := handler()
	if err != nil {
		log15.Error("Handler returned an error", "err", err)

		status = http.StatusInternalServerError
		payload = errorResponse{Error: err.Error()}
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log15.Error("Failed to serialize payload", "err", err)
		http.Error(w, fmt.Sprintf("Failed to serialize payload: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)

	if status != http.StatusNoContent {
		_, _ = io.Copy(w, bytes.NewReader(data))
	}
}
