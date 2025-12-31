package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vinimax001/finance-tracker/internal/finance"
)

func TestPOSTListSummary(t *testing.T) {
	svc := finance.NewService(finance.NewMemoryRepo())
	mux := NewMux(svc)

	// cria income
	body := []byte(`{"type":"income","category":"salary","amount_cents":500000}`)
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create income: %d %s", rec.Code, rec.Body.String())
	}

	// cria expense
	body = []byte(`{"type":"expense","category":"rent","amount_cents":150000}`)
	req = httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create expense: %d %s", rec.Code, rec.Body.String())
	}

	// lista período
	req = httptest.NewRequest(http.MethodGet, "/transactions?from=2025-11-01&to=2025-11-30", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}

	// resumo mensal
	req = httptest.NewRequest(http.MethodGet, "/summary/monthly?year=2025&month=11", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary: %d %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"income_cents":500000`)) {
		t.Fatalf("summary body: %s", rec.Body.String())
	}
}