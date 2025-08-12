package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
)

// InMemoryRateRepository stores the latest exchange rates in memory.
// It is safe for concurrent use.
type InMemoryRateRepository struct {
	mu          sync.RWMutex
	latestRates map[string]domain.ExchangeRate
}

// NewInMemoryRateRepository returns an empty in-memory repository instance.
func NewInMemoryRateRepository() *InMemoryRateRepository {
	return &InMemoryRateRepository{
		latestRates: make(map[string]domain.ExchangeRate),
	}
}

// UpdateLatestRates replaces the entire in-memory store with the provided set of rates.
func (r *InMemoryRateRepository) UpdateLatestRates(ctx context.Context, rates []domain.ExchangeRate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.latestRates = make(map[string]domain.ExchangeRate)
	for _, rate := range rates {
		key := fmt.Sprintf("%s_%s", rate.From, rate.To)
		r.latestRates[key] = rate
	}
	return nil
}

// GetLatestRate returns the most recently stored rate for the given pair.
func (r *InMemoryRateRepository) GetLatestRate(ctx context.Context, from, to domain.Currency) (domain.ExchangeRate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := fmt.Sprintf("%s_%s", from, to)
	rate, ok := r.latestRates[key]
	if !ok {
		return domain.ExchangeRate{}, fmt.Errorf("%w for %s to %s", ErrRateNotFound, from, to)
	}
	return rate, nil
}
