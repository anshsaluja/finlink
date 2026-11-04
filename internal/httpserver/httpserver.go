package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/anshsaluja/finlink/internal/connections"
	"github.com/anshsaluja/finlink/internal/financialdata"
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

type financialDataReader interface {
	AccountsByConnection(context.Context, string) ([]financialdata.Account, error)
	TransactionsByConnection(context.Context, string, int) ([]financialdata.Transaction, error)
}

type server struct {
	connections *connections.Service
	syncs       *syncs.Service
	data        financialDataReader
}

func New() http.Handler {
	registry := provider.NewRegistry(provider.MockNorthstar{}, provider.MockHarbor{})
	connectionService := connections.NewService(registry)
	syncService := syncs.NewService(syncs.NewMemoryRepository(), connectionService, &syncs.MemoryPublisher{})
	return NewWithServices(connectionService, syncService)
}

func NewWithServices(connectionService *connections.Service, syncService *syncs.Service) http.Handler {
	return newHandler(connectionService, syncService, nil)
}

func NewWithServicesAndData(connectionService *connections.Service, syncService *syncs.Service, data financialDataReader) http.Handler {
	return newHandler(connectionService, syncService, data)
}

func newHandler(connectionService *connections.Service, syncService *syncs.Service, data financialDataReader) http.Handler {
	s := &server{connections: connectionService, syncs: syncService, data: data}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /v1/connections", s.createConnection)
	mux.HandleFunc("GET /v1/connections/{id}", s.getConnection)
	mux.HandleFunc("POST /v1/connections/{id}/sync", s.requestSync)
	mux.HandleFunc("GET /v1/connections/{id}/accounts", s.getAccounts)
	mux.HandleFunc("GET /v1/connections/{id}/transactions", s.getTransactions)
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

func (s *server) getConnection(w http.ResponseWriter, r *http.Request) {
	connection, err := s.connections.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, connections.ErrConnectionNotFound) {
		writeError(w, http.StatusNotFound, "connection_not_found", "connection was not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read_failed", "connection could not be read")
		return
	}
	writeJSON(w, http.StatusOK, connection)
}

func (s *server) getAccounts(w http.ResponseWriter, r *http.Request) {
	if !s.connectionExists(w, r) {
		return
	}
	if s.data == nil {
		writeError(w, http.StatusServiceUnavailable, "data_unavailable", "financial data read service is unavailable")
		return
	}
	accounts, err := s.data.AccountsByConnection(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read_failed", "accounts could not be read")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": accounts})
}

func (s *server) getTransactions(w http.ResponseWriter, r *http.Request) {
	if !s.connectionExists(w, r) {
		return
	}
	if s.data == nil {
		writeError(w, http.StatusServiceUnavailable, "data_unavailable", "financial data read service is unavailable")
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be between 1 and 200")
			return
		}
		limit = parsed
	}
	transactions, err := s.data.TransactionsByConnection(r.Context(), r.PathValue("id"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read_failed", "transactions could not be read")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transactions": transactions})
}

func (s *server) connectionExists(w http.ResponseWriter, r *http.Request) bool {
	_, err := s.connections.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, connections.ErrConnectionNotFound) {
		writeError(w, http.StatusNotFound, "connection_not_found", "connection was not found")
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read_failed", "connection could not be read")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message})
}
