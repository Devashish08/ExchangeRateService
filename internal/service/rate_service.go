package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
	"github.com/Devashish08/ExchangeRateService/internal/metrics"
	"github.com/Devashish08/ExchangeRateService/internal/provider"
	"github.com/Devashish08/ExchangeRateService/internal/repository"
)

type RateService struct {
	fiatProvider   provider.FiatProvider
	cryptoProvider provider.CryptoProvider
	repo           repository.RateRepository
	baseCurrency   domain.Currency
	metrics        *metrics.Metrics
}

// NewRateService constructs a RateService with the provided providers, repository, and metrics.
func NewRateService(fiatP provider.FiatProvider, cryptoP provider.CryptoProvider, r repository.RateRepository, m *metrics.Metrics) *RateService {
	return &RateService{
		fiatProvider:   fiatP,
		cryptoProvider: cryptoP,
		repo:           r,
		baseCurrency:   domain.USD,
		metrics:        m,
	}
}

// RefreshRates fetches latest rates from providers, merges fiat and crypto, and updates the repository.
func (s *RateService) RefreshRates(ctx context.Context) {
	log.Println("Starting rate refresh for fiat and crypto...")

	fiatTargets := []domain.Currency{domain.INR, domain.EUR, domain.JPY, domain.GBP}
	fiatRates, err := s.fiatProvider.FetchLatestRates(ctx, s.baseCurrency, fiatTargets)
	if err != nil {
		log.Printf("ERROR: failed to fetch fiat rates: %v", err)
		s.metrics.ProviderRequestsTotal.WithLabelValues("fiat", "failure").Inc()
		return
	}
	s.metrics.ProviderRequestsTotal.WithLabelValues("fiat", "success").Inc()

	cryptoTargets := []domain.Currency{domain.BTC, domain.ETH, domain.USDT}
	cryptoRates, err := s.cryptoProvider.FetchCryptoRates(ctx, s.baseCurrency, cryptoTargets)
	if err != nil {
		log.Printf("ERROR: failed to fetch crypto rates: %v", err)
		s.metrics.ProviderRequestsTotal.WithLabelValues("crypto", "failure").Inc()
	} else {
		s.metrics.ProviderRequestsTotal.WithLabelValues("crypto", "success").Inc()
	}

	allBaseRates := append(fiatRates, cryptoRates...)
	if len(allBaseRates) == 0 {
		log.Printf("WARN: all providers returned zero base rates; skipping update")
		return
	}

	allPairs := s.calculateAllPairs(allBaseRates)
	err = s.repo.UpdateLatestRates(ctx, allPairs)
	if err != nil {
		log.Printf("ERROR: failed to update repository with new rates: %v", err)
	} else {
		log.Printf("Refreshed and stored %d rate pairs (including crypto).", len(allPairs))
	}
}

// ConvertAmount converts an amount between currencies using latest or historical rates.
func (s *RateService) ConvertAmount(ctx context.Context, amount float64, from, to domain.Currency, date *time.Time) (*domain.ConversionResult, error) {
	var finalRate float64

	if date != nil {
		if from.IsCrypto() || to.IsCrypto() {
			return nil, ErrHistoricalCrypto
		}

		if (*date).Before(time.Now().AddDate(0, 0, -90)) {
			return nil, ErrDateOutOfRange
		}

		if from == s.baseCurrency {
			rate, err := s.fiatProvider.FetchHistoricalRate(ctx, *date, from, to)
			if err != nil {
				return nil, fmt.Errorf("could not fetch historical rate for %s->%s: %w", from, to, err)
			}
			finalRate = rate.Rate
		} else {
			rateTo, err := s.fiatProvider.FetchHistoricalRate(ctx, *date, s.baseCurrency, to)
			if err != nil {
				return nil, fmt.Errorf("could not fetch historical base->to rate: %w", err)
			}
			rateFrom, err := s.fiatProvider.FetchHistoricalRate(ctx, *date, s.baseCurrency, from)
			if err != nil {
				return nil, fmt.Errorf("could not fetch historical base->from rate: %w", err)
			}
			finalRate = rateTo.Rate / rateFrom.Rate
		}
	} else {
		rate, err := s.repo.GetLatestRate(ctx, from, to)
		if err != nil {
			return nil, fmt.Errorf("could not get latest rate from cache: %w", err)
		}
		finalRate = rate.Rate
	}

	convertedAmount := amount * finalRate
	return &domain.ConversionResult{ConvertedAmount: convertedAmount}, nil
}

// calculateAllPairs derives all pairwise rates (fiat and crypto) from base rates.
func (s *RateService) calculateAllPairs(baseRates []domain.ExchangeRate) []domain.ExchangeRate {
	rateMap := make(map[domain.Currency]float64)
	rateMap[s.baseCurrency] = 1.0

	if len(baseRates) == 0 {
		return []domain.ExchangeRate{}
	}
	ratesDate := baseRates[0].Date

	for _, rate := range baseRates {
		rateMap[rate.To] = rate.Rate
	}

	var allRates []domain.ExchangeRate
	supportedCurrencies := []domain.Currency{
		domain.USD, domain.INR, domain.EUR, domain.JPY, domain.GBP,
		domain.BTC, domain.ETH, domain.USDT,
	}

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
