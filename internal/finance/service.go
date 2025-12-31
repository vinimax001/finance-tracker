package finance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrBadRequest = errors.New("bad request")
)

type Repository interface {
	Create(ctx context.Context, t *Transaction) error
	ListByPeriod(ctx context.Context, from, to time.Time) ([]Transaction, error)
	Delete(ctx context.Context, id uuid.UUID) error
	MonthlySummary(ctx context.Context, year int, month int) (*MonthlySummary, error)
}

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service { return &Service{repo: r} }

func (s *Service) Create(ctx context.Context, typ TxType, category string, amountCents int64, desc string) (*Transaction, error) {
	if typ != Income && typ != Expense {
		return nil, ErrBadRequest
	}
	category = strings.TrimSpace(category)
	if category == "" || amountCents <= 0 {
		return nil, ErrBadRequest
	}
	now := time.Now().UTC()
	tx := &Transaction{
		ID:          uuid.New(),
		Type:        typ,
		Category:    category,
		AmountCents: amountCents,
		OccurredAt:  now,
		Description: strings.TrimSpace(desc),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *Service) ListByPeriod(ctx context.Context, from, to time.Time) ([]Transaction, error) {
	if to.Before(from) {
		return nil, ErrBadRequest
	}
	return s.repo.ListByPeriod(ctx, from.UTC(), to.UTC())
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) MonthlySummary(ctx context.Context, year int, month int) (*MonthlySummary, error) {
	if month < 1 || month > 12 {
		return nil, ErrBadRequest
	}
	return s.repo.MonthlySummary(ctx, year, month)
}

// GenerateMonthlyReport cria um relatório textual formatado do mês
func (s *Service) GenerateMonthlyReport(ctx context.Context, year int, month int) (string, error) {
	summary, err := s.MonthlySummary(ctx, year, month)
	if err != nil {
		return "", err
	}

	// Formatar valores em reais
	incomeReais := float64(summary.Income) / 100.0
	expenseReais := float64(summary.Expense) / 100.0
	netReais := float64(summary.Net) / 100.0

	// Mapa de meses em português
	monthNames := map[int]string{
		1: "Janeiro", 2: "Fevereiro", 3: "Março", 4: "Abril",
		5: "Maio", 6: "Junho", 7: "Julho", 8: "Agosto",
		9: "Setembro", 10: "Outubro", 11: "Novembro", 12: "Dezembro",
	}
	monthName := monthNames[month]

	// Gerar relatório formatado
	report := fmt.Sprintf(`========================================
RELATÓRIO FINANCEIRO - %s/%d
========================================

Período: %s de %d
Total de Transações: %d

RESUMO FINANCEIRO:
------------------------------------------
Receitas:       R$ %.2f
Despesas:       R$ %.2f
------------------------------------------
Saldo Final:    R$ %.2f
------------------------------------------

`, monthName, year, monthName, year, summary.CountTx, incomeReais, expenseReais, netReais)

	if summary.FirstTxDate != "" {
		report += fmt.Sprintf("Primeira Transação: %s\n", summary.FirstTxDate)
	}
	if summary.LastTxDate != "" {
		report += fmt.Sprintf("Última Transação:   %s\n", summary.LastTxDate)
	}

	status := "POSITIVO ✓"
	if netReais < 0 {
		status = "NEGATIVO ✗"
	} else if netReais == 0 {
		status = "NEUTRO"
	}
	report += fmt.Sprintf("\nStatus: %s\n", status)
	report += "========================================\n"

	return report, nil
}