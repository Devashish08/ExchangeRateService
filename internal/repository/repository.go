package repository

import (
	"context"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
)

// RateRepository defines the persistence contract for storing and retrieving exchange rates.
type RateRepository interface {
	UpdateLatestRates(ctx context.Context, rates []domain.ExchangeRate) error
	GetLatestRate(ctx context.Context, from, to domain.Currency) (domain.ExchangeRate, error)
}
