package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONResponsesDisableCaching(t *testing.T) {
	tests := []struct {
		name string
		status int
		write func(http.ResponseWriter)
	}{
		{name: "success", status: http.StatusOK, write: func(w http.ResponseWriter) {
			writeJSON(w, http.StatusOK, map[string]any{"transactions": []string{"synthetic"}})
		}},
		{name: "error", status: http.StatusBadRequest, write: func(w http.ResponseWriter) {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid input")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.write(w)
			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d", w.Code, tt.status)
			}
			if got := w.Header().Get("Cache-Control"); got != "no-store" {
				t.Fatalf("Cache-Control = %q, want no-store", got)
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
		})
	}
}
