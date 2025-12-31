package finance

import (
	"context"
	"testing"
	"time"
)

func TestMemoryRepo_BasicFlow(t *testing.T) {
	r := NewMemoryRepo()
	s := NewService(r)

	_, err := s.Create(context.Background(), Income, "salary", 500000, "salário")
	if err != nil {
		t.Fatalf("create income: %v", err)
	}
	_, err = s.Create(context.Background(), Expense, "rent", 150000, "aluguel")
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}

	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now.Add(24 * time.Hour)
	list, err := s.ListByPeriod(context.Background(), from, to)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: err=%v len=%d", err, len(list))
	}

	sum, err := s.MonthlySummary(context.Background(), now.Year(), int(now.Month()))
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if sum.Income != 500000 || sum.Expense != 150000 || sum.Net != 350000 {
		t.Fatalf("summary mismatch: %+v", sum)
	}
}