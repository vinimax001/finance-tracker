package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/vinimax001/finance-tracker/internal/finance"
	"github.com/google/uuid"
)

const s3BucketName = "finance-tracker-releases"

func NewMux(svc *finance.Service) *http.ServeMux {
	m := http.NewServeMux()
	m.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ok(w, map[string]string{"status": "ok"})
	})
	m.HandleFunc("POST /transactions", postTransaction(svc))
	m.HandleFunc("GET /transactions", listTransactions(svc))
	m.HandleFunc("DELETE /transactions/{id}", deleteTransaction(svc))
	m.HandleFunc("GET /summary/monthly", monthlySummary(svc))
	m.HandleFunc("GET /reports/monthly", monthlyReport(svc))
	return m
}

type postTxReq struct {
	Type        string `json:"type"` // income | expense
	Category    string `json:"category"`
	AmountCents int64  `json:"amount_cents"` // centavos
	Description string `json:"description"`  // opcional
}

func postTransaction(svc *finance.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in postTxReq
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			serr(w, err, http.StatusBadRequest)
			return
		}
		tx, err := svc.Create(r.Context(), finance.TxType(in.Type), in.Category, in.AmountCents, in.Description)
		if err != nil {
			status := http.StatusInternalServerError
			if err == finance.ErrBadRequest {
				status = http.StatusBadRequest
			}
			serr(w, err, status)
			return
		}
		created(w, tx)
	}
}

func listTransactions(svc *finance.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		if fromStr == "" || toStr == "" {
			serr(w, errString("query params 'from' and 'to' are required (YYYY-MM-DD)"), http.StatusBadRequest)
			return
		}
		from, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			serr(w, err, http.StatusBadRequest)
			return
		}
		to, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			serr(w, err, http.StatusBadRequest)
			return
		}
		items, err := svc.ListByPeriod(r.Context(), from, to.Add(23*time.Hour+59*time.Minute+59*time.Second))
		if err != nil {
			status := http.StatusInternalServerError
			if err == finance.ErrBadRequest {
				status = http.StatusBadRequest
			}
			serr(w, err, status)
			return
		}
		ok(w, items)
	}
}

func deleteTransaction(svc *finance.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			serr(w, err, http.StatusBadRequest)
			return
		}
		if err := svc.Delete(r.Context(), id); err != nil {
			status := http.StatusInternalServerError
			if err == finance.ErrNotFound {
				status = http.StatusNotFound
			}
			serr(w, err, status)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func monthlySummary(svc *finance.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		yearStr := r.URL.Query().Get("year")
		monthStr := r.URL.Query().Get("month")
		if yearStr == "" || monthStr == "" {
			serr(w, errString("query params 'year' and 'month' are required"), http.StatusBadRequest)
			return
		}
		y, err := strconv.Atoi(yearStr)
		if err != nil {
			serr(w, err, http.StatusBadRequest)
			return
		}
		m, err := strconv.Atoi(monthStr)
		if err != nil {
			serr(w, err, http.StatusBadRequest)
			return
		}
		sum, err := svc.MonthlySummary(r.Context(), y, m)
		if err != nil {
			status := http.StatusInternalServerError
			if err == finance.ErrBadRequest {
				status = http.StatusBadRequest
			}
			serr(w, err, status)
			return
		}
		ok(w, sum)
	}
}

func monthlyReport(svc *finance.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		yearStr := r.URL.Query().Get("year")
		monthStr := r.URL.Query().Get("month")
		if yearStr == "" || monthStr == "" {
			serr(w, errString("query params 'year' and 'month' are required"), http.StatusBadRequest)
			return
		}
		y, err := strconv.Atoi(yearStr)
		if err != nil {
			serr(w, err, http.StatusBadRequest)
			return
		}
		m, err := strconv.Atoi(monthStr)
		if err != nil {
			serr(w, err, http.StatusBadRequest)
			return
		}

		// Gerar relatório textual
		reportText, err := svc.GenerateMonthlyReport(r.Context(), y, m)
		if err != nil {
			status := http.StatusInternalServerError
			if err == finance.ErrBadRequest {
				status = http.StatusBadRequest
			}
			serr(w, err, status)
			return
		}

		// Nome do arquivo: report-YYYY-MM.txt
		fileName := fmt.Sprintf("reports/report-%04d-%02d.txt", y, m)

		// Upload para S3 em background com contexto independente
		go func() {
			// Criar contexto com timeout para o upload (não vinculado à requisição HTTP)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := finance.UploadReportToS3(ctx, s3BucketName, fileName, reportText); err != nil {
				// Log do erro (em produção usaria logger apropriado)
				fmt.Printf("Error uploading report to S3: %v\n", err)
			} else {
				fmt.Printf("Report uploaded to S3: s3://%s/%s\n", s3BucketName, fileName)
			}
		}()

		// Buscar summary para resposta JSON
		sum, err := svc.MonthlySummary(r.Context(), y, m)
		if err != nil {
			status := http.StatusInternalServerError
			if err == finance.ErrBadRequest {
				status = http.StatusBadRequest
			}
			serr(w, err, status)
			return
		}

		// Resposta com summary JSON + informação do arquivo S3
		type reportResp struct {
			*finance.MonthlySummary
			ReportText string `json:"report_text"`
			S3File     string `json:"s3_file"`
		}

		ok(w, reportResp{
			MonthlySummary: sum,
			ReportText:     reportText,
			S3File:         fmt.Sprintf("s3://%s/%s", s3BucketName, fileName),
		})
	}
}

// utilitários de resposta JSON
type stringErr string

func (e stringErr) Error() string { return string(e) }
func errString(s string) error    { return stringErr(s) }

func ok(w http.ResponseWriter, v any)      { writeJSON(w, http.StatusOK, v) }
func created(w http.ResponseWriter, v any) { writeJSON(w, http.StatusCreated, v) }
func serr(w http.ResponseWriter, err error, status int) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}