package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
	"github.com/Devashish08/ExchangeRateService/internal/metrics"
	"github.com/Devashish08/ExchangeRateService/internal/provider"
	"github.com/Devashish08/ExchangeRateService/internal/repository"
)

type Config struct {
	BaseCurrency  domain.Currency
	FiatTargets   []domain.Currency
	CryptoTargets []domain.Currency
}

type RateService struct {
	fiatProvider   provider.FiatProvider
	cryptoProvider provider.CryptoProvider
	repo           repository.RateRepository
	config         Config
	logger         *slog.Logger
	metrics        *metrics.Metrics
}

func NewRateService(
	fiatP provider.FiatProvider,
	cryptoP provider.CryptoProvider,
	r repository.RateRepository,
	cfg Config,
	l *slog.Logger,
	m *metrics.Metrics,
) *RateService {
	return &RateService{
		fiatProvider:   fiatP,
		cryptoProvider: cryptoP,
		repo:           r,
		config:         cfg,
		logger:         l,
		metrics:        m,
	}
}

func (s *RateService) RefreshRates(ctx context.Context) {
	log := s.logger.With("operation", "RefreshRates")
	log.Info("Starting rate refresh for fiat and crypto...")

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allBaseRates []domain.ExchangeRate

	wg.Add(1)
	go func() {
		defer wg.Done()
		fiatRates, err := s.fiatProvider.FetchLatestRates(ctx, s.config.BaseCurrency, s.config.FiatTargets)
		if err != nil {
			log.Error("failed to fetch fiat rates", "error", err)
			s.metrics.ProviderRequestsTotal.WithLabelValues("fiat", "failure").Inc()
			return
		}
		s.metrics.ProviderRequestsTotal.WithLabelValues("fiat", "success").Inc()
		mu.Lock()
		allBaseRates = append(allBaseRates, fiatRates...)
		mu.Unlock()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		cryptoRates, err := s.cryptoProvider.FetchCryptoRates(ctx, s.config.BaseCurrency, s.config.CryptoTargets)
		if err != nil {
			log.Error("failed to fetch crypto rates", "error", err)
			s.metrics.ProviderRequestsTotal.WithLabelValues("crypto", "failure").Inc()
			return
		}
		s.metrics.ProviderRequestsTotal.WithLabelValues("crypto", "success").Inc()
		mu.Lock()
		allBaseRates = append(allBaseRates, cryptoRates...)
		mu.Unlock()
	}()

	wg.Wait()

	log.Info("Finished fetching from all providers.", "successful_rates_count", len(allBaseRates))

	if len(allBaseRates) == 0 {
		log.Warn("All providers returned zero base rates; skipping cache update.")
		return
	}

	allPairs := s.calculateAllPairs(allBaseRates)
	err := s.repo.UpdateLatestRates(ctx, allPairs)
	if err != nil {
		log.Error("failed to update repository with new rates", "error", err)
	} else {
		log.Info("Refreshed and stored rate pairs.", "count", len(allPairs))
	}
}

func (s *RateService) ConvertAmount(ctx context.Context, amount float64, from, to domain.Currency, date *time.Time) (*domain.ConversionResult, error) {
	var finalRate float64
	var err error

	if date != nil {
		finalRate, err = s.getHistoricalRate(ctx, *date, from, to)
	} else {
		var rate domain.ExchangeRate
		rate, err = s.repo.GetLatestRate(ctx, from, to)
		if err == nil {
			finalRate = rate.Rate
		}
	}

	if err != nil {
		return nil, err
	}

	return &domain.ConversionResult{ConvertedAmount: amount * finalRate}, nil
}

func (s *RateService) getHistoricalRate(ctx context.Context, date time.Time, from, to domain.Currency) (float64, error) {
	if from.IsCrypto() || to.IsCrypto() {
		return 0, ErrHistoricalCrypto
	}
	if date.Before(time.Now().UTC().AddDate(0, 0, -90)) {
		return 0, ErrDateOutOfRange
	}

	requiredTargets := []domain.Currency{from, to}
	if from == s.config.BaseCurrency || to == s.config.BaseCurrency {
		requiredTargets = []domain.Currency{from, to}
	}

	rates, err := s.fiatProvider.FetchHistoricalRates(ctx, date, s.config.BaseCurrency, requiredTargets)
	if err != nil {
		return 0, fmt.Errorf("could not fetch historical rates from provider: %w", err)
	}

	rateMap := make(map[domain.Currency]float64)
	rateMap[s.config.BaseCurrency] = 1.0
	for _, rate := range rates {
		rateMap[rate.To] = rate.Rate
	}

	fromRate, fromOk := rateMap[from]
	toRate, toOk := rateMap[to]
	if !fromOk || !toOk {
		return 0, fmt.Errorf("provider did not return all required historical rates for %s and %s", from, to)
	}

	if fromRate == 0 {
		return 0, fmt.Errorf("base rate for %s is zero, cannot perform division", from)
	}

	return toRate / fromRate, nil
}

func (s *RateService) calculateAllPairs(baseRates []domain.ExchangeRate) []domain.ExchangeRate {
	if len(baseRates) == 0 {
		return []domain.ExchangeRate{}
	}

	rateMap := make(map[domain.Currency]float64)
	rateMap[s.config.BaseCurrency] = 1.0

	ratesDate := baseRates[0].Date

	for _, rate := range baseRates {
		rateMap[rate.To] = rate.Rate
	}

	var supportedCurrencies []domain.Currency
	for curr := range rateMap {
		supportedCurrencies = append(supportedCurrencies, curr)
	}

	var allRates []domain.ExchangeRate
	for _, from := range supportedCurrencies {
		for _, to := range supportedCurrencies {
			if from == to {
				continue
			}
			conversionRate := rateMap[to] / rateMap[from]
			allRates = append(allRates, domain.ExchangeRate{
				From: from, To: to, Rate: conversionRate, Date: ratesDate,
			})
		}
	}
	return allRates
}
