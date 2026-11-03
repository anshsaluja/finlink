package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/anshsaluja/finlink/internal/connections"
	"github.com/anshsaluja/finlink/internal/provider"
	"github.com/anshsaluja/finlink/internal/syncs"
)

type healthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

type createConnectionRequest struct {
	Provider    string               `json:"provider"`
	Credentials provider.Credentials `json:"credentials"`
}

type server struct {
	connections *connections.Service
	syncs       *syncs.Service
}

func New() http.Handler {
	registry := provider.NewRegistry(provider.MockNorthstar{}, provider.MockHarbor{})
	connectionService := connections.NewService(registry)
	syncService := syncs.NewService(syncs.NewMemoryRepository(), connectionService, &syncs.MemoryPublisher{})
	return NewWithServices(connectionService, syncService)
}

func NewWithServices(connectionService *connections.Service, syncService *syncs.Service) http.Handler {
	s := &server{connections: connectionService, syncs: syncService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /v1/connections", s.createConnection)
	mux.HandleFunc("POST /v1/connections/{id}/sync", s.requestSync)
	mux.HandleFunc("GET /v1/sync-runs/{id}", s.getSyncRun)
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Service: "finlink-api", Timestamp: time.Now().UTC().Format(time.RFC3339)})
}

func (s *server) createConnection(w http.ResponseWriter, r *http.Request) {
	var request createConnectionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || request.Provider == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "provider and credentials are required")
		return
	}
	connection, err := s.connections.Create(r.Context(), request.Provider, request.Credentials)
	switch {
	case errors.Is(err, connections.ErrProviderNotFound):
		writeError(w, http.StatusNotFound, "provider_not_found", "provider is not supported")
	case errors.Is(err, provider.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "provider credentials are invalid")
	case err != nil:
		writeError(w, http.StatusBadGateway, "provider_error", "provider connection failed")
	default:
		writeJSON(w, http.StatusCreated, connection)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message})
}
