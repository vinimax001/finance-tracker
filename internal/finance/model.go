package finance

import (
	"time"

	"github.com/google/uuid"
)

type TxType string

const (
	Income  TxType = "income"
	Expense TxType = "expense"
)

type Transaction struct {
	ID          uuid.UUID `json:"id"`
	Type        TxType    `json:"type"`         // "income" | "expense"
	Category    string    `json:"category"`     // ex: salary, rent, food
	AmountCents int64     `json:"amount_cents"` // ex: 12345 = R$ 123,45
	OccurredAt  time.Time `json:"occurred_at"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MonthlySummary struct {
	Year        int   `json:"year"`
	Month       int   `json:"month"`
	Income      int64 `json:"income_cents"`
	Expense     int64 `json:"expense_cents"`
	Net         int64 `json:"net_cents"`
	CountTx     int   `json:"count_transactions"`
	FirstTxDate string `json:"first_tx,omitempty"`
	LastTxDate  string `json:"last_tx,omitempty"`
}