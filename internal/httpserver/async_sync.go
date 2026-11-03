package httpserver

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/anshsaluja/finlink/internal/connections"
	"github.com/anshsaluja/finlink/internal/syncs"
)

// requestSync persists a durable sync request before asynchronous processing.
func (s *server) requestSync(w http.ResponseWriter, r *http.Request) {
	run, err := s.syncs.Request(r.Context(), r.PathValue("id"), r.Header.Get("Idempotency-Key"))
	switch {
	case errors.Is(err, connections.ErrConnectionNotFound):
		writeError(w, http.StatusNotFound, "connection_not_found", "connection does not exist")
	case err != nil:
		writeError(w, http.StatusServiceUnavailable, "sync_request_failed", "sync request could not be queued")
	default:
		w.Header().Set("Location", "/v1/sync-runs/"+strconv.FormatInt(run.ID, 10))
		writeJSON(w, http.StatusAccepted, run)
	}
}

func (s *server) getSyncRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_sync_run_id", "sync run id must be a positive integer")
		return
	}
	run, err := s.syncs.Get(r.Context(), id)
	switch {
	case errors.Is(err, syncs.ErrRunNotFound):
		writeError(w, http.StatusNotFound, "sync_run_not_found", "sync run does not exist")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "sync_run_read_failed", "sync run could not be read")
	default:
		writeJSON(w, http.StatusOK, run)
	}
}
