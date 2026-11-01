package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConnectionCreateAndSync(t *testing.T) {
	handler := New()
	request := httptest.NewRequest(http.MethodPost, "/v1/connections", strings.NewReader(`{"provider":"northstar","credentials":{"username":"demo","password":"secret"}}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", response.Code, response.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("expected connection id")
	}

	syncRequest := httptest.NewRequest(http.MethodPost, "/v1/connections/"+created.ID+"/sync", nil)
	syncResponse := httptest.NewRecorder()
	handler.ServeHTTP(syncResponse, syncRequest)
	if syncResponse.Code != http.StatusOK {
		t.Fatalf("sync status = %d body=%s", syncResponse.Code, syncResponse.Body.String())
	}
	var synced struct {
		Transactions []any `json:"transactions"`
	}
	if err := json.NewDecoder(syncResponse.Body).Decode(&synced); err != nil {
		t.Fatal(err)
	}
	if len(synced.Transactions) != 1 {
		t.Fatalf("transactions = %d", len(synced.Transactions))
	}
}
