// In file: internal/service/rate_service_test.go

package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
	"github.com/Devashish08/ExchangeRateService/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockFiatProvider struct {
	mock.Mock
}

func (m *MockFiatProvider) FetchLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) ([]domain.ExchangeRate, error) {
	args := m.Called(ctx, base, targets)
	return args.Get(0).([]domain.ExchangeRate), args.Error(1)
}

func (m *MockFiatProvider) FetchHistoricalRates(ctx context.Context, date time.Time, base domain.Currency, targets []domain.Currency) ([]domain.ExchangeRate, error) {
	args := m.Called(ctx, date, base, targets)
	return args.Get(0).([]domain.ExchangeRate), args.Error(1)
}

type MockCryptoProvider struct {
	mock.Mock
}

func (m *MockCryptoProvider) FetchCryptoRates(ctx context.Context, baseFiat domain.Currency, cryptos []domain.Currency) ([]domain.ExchangeRate, error) {
	args := m.Called(ctx, baseFiat, cryptos)
	return args.Get(0).([]domain.ExchangeRate), args.Error(1)
}

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) UpdateLatestRates(ctx context.Context, rates []domain.ExchangeRate) error {
	args := m.Called(ctx, rates)
	return args.Error(0)
}

func (m *MockRepository) GetLatestRate(ctx context.Context, from, to domain.Currency) (domain.ExchangeRate, error) {
	args := m.Called(ctx, from, to)
	return args.Get(0).(domain.ExchangeRate), args.Error(1)
}

type testSetup struct {
	service    *RateService
	mockFiat   *MockFiatProvider
	mockCrypto *MockCryptoProvider
	mockRepo   *MockRepository
	logger     *slog.Logger
	metrics    *metrics.Metrics
	config     Config
}

func newTestSetup() testSetup {
	mockFiat := new(MockFiatProvider)
	mockCrypto := new(MockCryptoProvider)
	mockRepo := new(MockRepository)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics(reg)

	config := Config{
		BaseCurrency:  domain.USD,
		FiatTargets:   []domain.Currency{domain.EUR},
		CryptoTargets: []domain.Currency{domain.BTC},
	}

	service := NewRateService(mockFiat, mockCrypto, mockRepo, config, logger, m)

	return testSetup{
		service:    service,
		mockFiat:   mockFiat,
		mockCrypto: mockCrypto,
		mockRepo:   mockRepo,
		logger:     logger,
		metrics:    m,
		config:     config,
	}
}

func TestRefreshRates_Success(t *testing.T) {
	ts := newTestSetup()

	ts.mockFiat.On("FetchLatestRates", mock.Anything, ts.config.BaseCurrency, ts.config.FiatTargets).Return([]domain.ExchangeRate{{From: "USD", To: "EUR", Rate: 0.9}}, nil)
	ts.mockCrypto.On("FetchCryptoRates", mock.Anything, ts.config.BaseCurrency, ts.config.CryptoTargets).Return([]domain.ExchangeRate{{From: "USD", To: "BTC", Rate: 50000}}, nil)
	ts.mockRepo.On("UpdateLatestRates", mock.Anything, mock.AnythingOfType("[]domain.ExchangeRate")).Return(nil)

	ts.service.RefreshRates(context.Background())

	ts.mockFiat.AssertExpectations(t)
	ts.mockCrypto.AssertExpectations(t)
	ts.mockRepo.AssertExpectations(t)
}

func TestRefreshRates_PartialFailure(t *testing.T) {
	ts := newTestSetup()

	ts.mockFiat.On("FetchLatestRates", mock.Anything, mock.Anything, mock.Anything).Return([]domain.ExchangeRate{}, errors.New("fiat provider down"))
	ts.mockCrypto.On("FetchCryptoRates", mock.Anything, mock.Anything, mock.Anything).Return([]domain.ExchangeRate{{From: "USD", To: "BTC", Rate: 50000}}, nil)
	ts.mockRepo.On("UpdateLatestRates", mock.Anything, mock.AnythingOfType("[]domain.ExchangeRate")).Return(nil)

	ts.service.RefreshRates(context.Background())

	ts.mockCrypto.AssertExpectations(t)
	ts.mockRepo.AssertExpectations(t)
}

func TestConvertAmount_Latest(t *testing.T) {
	ts := newTestSetup()
	ts.mockRepo.On("GetLatestRate", mock.Anything, domain.USD, domain.EUR).Return(domain.ExchangeRate{Rate: 0.9}, nil)

	result, err := ts.service.ConvertAmount(context.Background(), 100, domain.USD, domain.EUR, nil)

	assert.NoError(t, err)
	assert.Equal(t, 90.0, result.ConvertedAmount)
	ts.mockRepo.AssertExpectations(t)
}

func TestConvertAmount_Historical(t *testing.T) {
	ts := newTestSetup()
	historicalDate := time.Now().UTC().AddDate(0, 0, -10)
	ts.mockFiat.On("FetchHistoricalRates", mock.Anything, historicalDate, domain.USD, []domain.Currency{domain.EUR, domain.JPY}).Return([]domain.ExchangeRate{
		{From: "USD", To: "EUR", Rate: 0.9},
		{From: "USD", To: "JPY", Rate: 150},
	}, nil)

	result, err := ts.service.ConvertAmount(context.Background(), 100, domain.EUR, domain.JPY, &historicalDate)

	assert.NoError(t, err)
	assert.InDelta(t, 16666.67, result.ConvertedAmount, 0.01)
	ts.mockFiat.AssertExpectations(t)
}

func TestConvertAmount_Errors(t *testing.T) {
	ts := newTestSetup()

	oldDate := time.Now().UTC().AddDate(0, 0, -91)
	_, err := ts.service.ConvertAmount(context.Background(), 100, domain.USD, domain.EUR, &oldDate)
	assert.ErrorIs(t, err, ErrDateOutOfRange)

	validDate := time.Now().UTC().AddDate(0, 0, -10)
	_, err = ts.service.ConvertAmount(context.Background(), 100, domain.USD, domain.BTC, &validDate)
	assert.ErrorIs(t, err, ErrHistoricalCrypto)
}
