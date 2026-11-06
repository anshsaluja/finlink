package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anshsaluja/finlink/internal/financialdata"
)

// A short page must terminate pagination, even when the requested offset is nonzero.
func TestTransactionPaginationShortPageStopsContinuation(t *testing.T) {
	reader := &shortPageReader{}
	handler, path := transactionHandler(t, reader)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path+"?limit=3&offset=7", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var body struct {
		Transactions []financialdata.Transaction `json:"transactions"`
		Limit        int                         `json:"limit"`
		Offset       int                         `json:"offset"`
		MayHaveMore  bool                        `json:"may_have_more"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Transactions) != 1 || body.Limit != 3 || body.Offset != 7 || body.MayHaveMore {
		t.Fatalf("unexpected short page: %+v", body)
	}
	if reader.limit != 3 || reader.offset != 7 {
		t.Fatalf("repository received limit=%d offset=%d", reader.limit, reader.offset)
	}
}

type shortPageReader struct {
	paginatedReader
	limit  int
	offset int
}

func (r *shortPageReader) TransactionsByConnectionPage(_ context.Context, _ string, limit, offset int) ([]financialdata.Transaction, error) {
	r.limit, r.offset = limit, offset
	return []financialdata.Transaction{{}}, nil
}
