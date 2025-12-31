package finance

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type pgRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) Repository { return &pgRepo{db: db} }

func (p *pgRepo) Create(ctx context.Context, t *Transaction) error {
	const q = `
		INSERT INTO transactions (id, type, category, amount_cents, occurred_at, description, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`
	_, err := p.db.ExecContext(ctx, q,
		t.ID, t.Type, t.Category, t.AmountCents, t.OccurredAt, t.Description, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (p *pgRepo) ListByPeriod(ctx context.Context, from, to time.Time) ([]Transaction, error) {
	const q = `
		SELECT id, type, category, amount_cents, occurred_at, description, created_at, updated_at
		FROM transactions
		WHERE occurred_at >= $1 AND occurred_at <= $2
		ORDER BY occurred_at ASC, created_at ASC
	`
	rows, err := p.db.QueryContext(ctx, q, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Type, &t.Category, &t.AmountCents, &t.OccurredAt, &t.Description, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *pgRepo) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := p.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *pgRepo) MonthlySummary(ctx context.Context, year int, month int) (*MonthlySummary, error) {
	const q = `
		WITH m AS (
			SELECT
				SUM(CASE WHEN type = 'income'  THEN amount_cents ELSE 0 END) AS income,
				SUM(CASE WHEN type = 'expense' THEN amount_cents ELSE 0 END) AS expense,
				COUNT(*) AS cnt,
				MIN(occurred_at) AS first_tx,
				MAX(occurred_at) AS last_tx
			FROM transactions
			WHERE EXTRACT(YEAR FROM occurred_at) = $1
			  AND EXTRACT(MONTH FROM occurred_at) = $2
		)
		SELECT COALESCE(income,0), COALESCE(expense,0), COALESCE(cnt,0), first_tx, last_tx FROM m;
	`
	var inc, exp int64
	var cnt int
	var first, last sql.NullTime
	if err := p.db.QueryRowContext(ctx, q, year, month).Scan(&inc, &exp, &cnt, &first, &last); err != nil {
		return nil, err
	}
	ms := &MonthlySummary{
		Year:    year,
		Month:   month,
		Income:  inc,
		Expense: exp,
		Net:     inc - exp,
		CountTx: cnt,
	}
	if first.Valid {
		ms.FirstTxDate = first.Time.Format(time.RFC3339)
	}
	if last.Valid {
		ms.LastTxDate = last.Time.Format(time.RFC3339)
	}
	return ms, nil
}