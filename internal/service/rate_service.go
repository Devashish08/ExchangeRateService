package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Devahish08/ExchangeRateService/internal/domain"
	"github.com/Devahish08/ExchangeRateService/internal/provider"
	"github.com/Devahish08/ExchangeRateService/internal/repository"
)

// RateService orchestrates fetching exchange rates from a provider and storing
// them in a repository.
type RateService struct {
	provider     *provider.ExchangeRateHostProvider
	repo         repository.RateRepository
	baseCurrency domain.Currency
}

// NewRateService returns a RateService configured with the given provider and
// repository. The base currency is fixed to USD.
func NewRateService(p *provider.ExchangeRateHostProvider, r repository.RateRepository) *RateService {
	return &RateService{
		provider:     p,
		repo:         r,
		baseCurrency: domain.USD,
	}
}

// RefreshRates fetches the latest rates and updates the repository. It is safe
// to call periodically from a scheduler.
func (s *RateService) RefreshRates(ctx context.Context) {
	log.Println("Starting rate refresh")

	targets := []domain.Currency{domain.INR, domain.EUR, domain.JPY, domain.GBP}

	baseRates, err := s.provider.FetchLatestRates(ctx, s.baseCurrency, targets)
	if err != nil {
		log.Printf("ERROR: fetch latest rates: %v", err)
		return
	}

	if len(baseRates) == 0 {
		log.Printf("WARN: provider returned zero base rates; skipping update")
		return
	}

	allRates := s.calculateAllPairs(baseRates)

	err = s.repo.UpdateLatestRates(ctx, allRates)
	if err != nil {
		log.Printf("ERROR: update repository with new rates: %v", err)
	} else {
		log.Printf("Refreshed and stored %d rate pairs", len(allRates))
	}
}

// calculateAllPairs derives rates for all supported currency pairs using the
// base-currency rates via cross rates.
func (s *RateService) calculateAllPairs(baseRates []domain.ExchangeRate) []domain.ExchangeRate {
	rateMap := make(map[domain.Currency]float64)
	rateMap[s.baseCurrency] = 1.0

	ratesDate := baseRates[0].Date

	for _, rate := range baseRates {
		rateMap[rate.To] = rate.Rate
	}

	var allRates []domain.ExchangeRate
	supportedCurrencies := []domain.Currency{domain.USD, domain.INR, domain.EUR, domain.JPY, domain.GBP}

	for _, from := range supportedCurrencies {
		for _, to := range supportedCurrencies {
			if from == to {
				continue
			}

			conversionRate := rateMap[to] / rateMap[from]

			allRates = append(allRates, domain.ExchangeRate{
				From: from,
				To:   to,
				Rate: conversionRate,
				Date: ratesDate,
			})
		}
	}
	return allRates
}

func (s *RateService) ConvertAmount(ctx context.Context, amount float64, from, to domain.Currency, date *time.Time) (*domain.ConversionResult, error) {
	var finalRate float64

	if date != nil {
		// --- Historical Conversion ---
		// Enforce 90-day lookback limit
		if (*date).Before(time.Now().AddDate(0, 0, -90)) {
			return nil, fmt.Errorf("date is beyond the 90-day historical data limit")
		}

		// Fetch historical rate. We may need to do a cross-currency calculation.
		// To get FROM -> TO, we calculate (BASE -> TO) / (BASE -> FROM)
		if from == s.baseCurrency {
			rate, err := s.provider.FetchHistoricalRate(ctx, *date, from, to)
			if err != nil {
				return nil, fmt.Errorf("could not fetch historical rate for %s->%s: %w", from, to, err)
			}
			finalRate = rate.Rate
		} else {
			// Fetch both 'from' and 'to' rates relative to the base currency (USD)
			rateTo, err := s.provider.FetchHistoricalRate(ctx, *date, s.baseCurrency, to)
			if err != nil {
				return nil, fmt.Errorf("could not fetch historical base->to rate: %w", err)
			}
			rateFrom, err := s.provider.FetchHistoricalRate(ctx, *date, s.baseCurrency, from)
			if err != nil {
				return nil, fmt.Errorf("could not fetch historical base->from rate: %w", err)
			}
			finalRate = rateTo.Rate / rateFrom.Rate
		}
	} else {
		// --- Latest Rate Conversion ---
		rate, err := s.repo.GetLatestRate(ctx, from, to)
		if err != nil {
			return nil, fmt.Errorf("could not get latest rate from cache: %w", err)
		}
		finalRate = rate.Rate
	}

	convertedAmount := amount * finalRate
	return &domain.ConversionResult{ConvertedAmount: convertedAmount}, nil
}
