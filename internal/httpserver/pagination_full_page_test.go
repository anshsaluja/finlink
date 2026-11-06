package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anshsaluja/finlink/internal/financialdata"
)

// A full page is only a continuation hint; the client must request another page.
func TestTransactionPaginationFullPageContinuationHint(t *testing.T) {
	reader := &fullPageReader{}
	handler, path := transactionHandler(t, reader)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path+"?limit=1&offset=5", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	records, ok := body["transactions"].([]any)
	if !ok || len(records) != 1 || body["limit"] != float64(1) || body["offset"] != float64(5) || body["may_have_more"] != true {
		t.Fatalf("unexpected page: %+v", body)
	}
	if reader.offset != 5 || reader.limit != 1 {
		t.Fatalf("wrong repository page: limit=%d offset=%d", reader.limit, reader.offset)
	}
}

type fullPageReader struct {
	paginatedReader
	limit  int
	offset int
}

func (p *fullPageReader) TransactionsByConnectionPage(_ context.Context, _ string, limit, offset int) ([]financialdata.Transaction, error) {
	p.limit = limit
	p.offset = offset
	return []financialdata.Transaction{{}}, nil
}
