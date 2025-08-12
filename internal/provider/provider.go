package provider

import (
	"context"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
)

// FiatProvider defines operations for fetching fiat exchange rates.
type FiatProvider interface {
	FetchLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) ([]domain.ExchangeRate, error)
	FetchHistoricalRates(ctx context.Context, date time.Time, base domain.Currency, targets []domain.Currency) ([]domain.ExchangeRate, error)
}

// CryptoProvider defines operations for fetching cryptocurrency exchange rates.
type CryptoProvider interface {
	FetchCryptoRates(ctx context.Context, baseFiat domain.Currency, cryptos []domain.Currency) ([]domain.ExchangeRate, error)
}
