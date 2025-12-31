package finance

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

type memoryRepo struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*Transaction
}

func NewMemoryRepo() Repository {
	return &memoryRepo{data: make(map[uuid.UUID]*Transaction)}
}

func (m *memoryRepo) Create(ctx context.Context, t *Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *t
	m.data[t.ID] = &cp
	return nil
}

func (m *memoryRepo) ListByPeriod(ctx context.Context, from, to time.Time) ([]Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Transaction
	for _, v := range m.data {
		if !v.OccurredAt.Before(from) && !v.OccurredAt.After(to) {
			out = append(out, *v)
		}
	}
	slices.SortFunc(out, func(a, b Transaction) int {
		return a.OccurredAt.Compare(b.OccurredAt)
	})
	return out, nil
}

func (m *memoryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[id]; !ok {
		return ErrNotFound
	}
	delete(m.data, id)
	return nil
}

func (m *memoryRepo) MonthlySummary(ctx context.Context, year int, month int) (*MonthlySummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var inc, exp int64
	var cnt int
	var first, last *time.Time
	for _, v := range m.data {
		if v.OccurredAt.Year() == year && int(v.OccurredAt.Month()) == month {
			if v.Type == Income {
				inc += v.AmountCents
			} else if v.Type == Expense {
				exp += v.AmountCents
			}
			cnt++
			if first == nil || v.OccurredAt.Before(*first) {
				t := v.OccurredAt
				first = &t
			}
			if last == nil || v.OccurredAt.After(*last) {
				t := v.OccurredAt
				last = &t
			}
		}
	}
	ms := &MonthlySummary{Year: year, Month: month, Income: inc, Expense: exp, Net: inc - exp, CountTx: cnt}
	if first != nil {
		ms.FirstTxDate = first.Format(time.RFC3339)
		ms.LastTxDate = last.Format(time.RFC3339)
	}
	return ms, nil
}