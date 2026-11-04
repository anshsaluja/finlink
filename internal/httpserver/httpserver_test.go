package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConnectionCreateAndRequestSync(t *testing.T) {
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
	syncRequest.Header.Set("Idempotency-Key", "demo-request-1")
	syncResponse := httptest.NewRecorder()
	handler.ServeHTTP(syncResponse, syncRequest)
	if syncResponse.Code != http.StatusAccepted {
		t.Fatalf("sync status = %d body=%s", syncResponse.Code, syncResponse.Body.String())
	}
	var run struct {
		ID             int64  `json:"id"`
		Status         string `json:"status"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(syncResponse.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.ID == 0 || run.Status != "requested" || run.IdempotencyKey != "demo-request-1" {
		t.Fatalf("unexpected sync run: %+v", run)
	}
	if syncResponse.Header().Get("Location") == "" {
		t.Fatal("expected sync run location")
	}

	statusRequest := httptest.NewRequest(http.MethodGet, syncResponse.Header().Get("Location"), nil)
	statusResponse := httptest.NewRecorder()
	handler.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("status code = %d body=%s", statusResponse.Code, statusResponse.Body.String())
	}
}

func TestSyncRequestIsIdempotent(t *testing.T) {
	handler := New()
	create := httptest.NewRequest(http.MethodPost, "/v1/connections", strings.NewReader(`{"provider":"harbor","credentials":{"token":"harbor-demo-token"}}`))
	createdResponse := httptest.NewRecorder()
	handler.ServeHTTP(createdResponse, create)
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createdResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	var firstID int64
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/connections/"+created.ID+"/sync", nil)
		req.Header.Set("Idempotency-Key", "same-request")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusAccepted {
			t.Fatalf("attempt %d status=%d body=%s", i, res.Code, res.Body.String())
		}
		var run struct {
			ID int64 `json:"id"`
		}
		if err := json.NewDecoder(res.Body).Decode(&run); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			firstID = run.ID
		} else if run.ID != firstID {
			t.Fatalf("idempotent request created run %d after %d", run.ID, firstID)
		}
	}
}
