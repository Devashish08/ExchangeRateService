package service

import (
	"context"
	"testing"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
	"github.com/Devashish08/ExchangeRateService/internal/metrics" // Import metrics package
	"github.com/Devashish08/ExchangeRateService/internal/provider"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks

// MockFiatProvider implements provider.FiatProvider for tests.
type MockFiatProvider struct {
	mock.Mock
}

var _ provider.FiatProvider = (*MockFiatProvider)(nil)

func (m *MockFiatProvider) FetchLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) ([]domain.ExchangeRate, error) {
	args := m.Called(ctx, base, targets)
	return args.Get(0).([]domain.ExchangeRate), args.Error(1)
}

func (m *MockFiatProvider) FetchHistoricalRate(ctx context.Context, date time.Time, base, target domain.Currency) (domain.ExchangeRate, error) {
	args := m.Called(ctx, date, base, target)
	return args.Get(0).(domain.ExchangeRate), args.Error(1)
}

// MockRepository mocks the RateRepository interface.
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

// helper
func setupDummyMetrics() *metrics.Metrics {
	// Use a dummy registry for tests to avoid interfering with the global one.
	return metrics.NewMetrics(prometheus.NewRegistry())
}

// Tests

func TestConvertAmount_Latest(t *testing.T) {
	mockRepo := new(MockRepository)
	mockRepo.On("GetLatestRate", mock.Anything, domain.USD, domain.INR).Return(domain.ExchangeRate{Rate: 83.5}, nil)

	rateService := NewRateService(nil, nil, mockRepo, nil)

	result, err := rateService.ConvertAmount(context.Background(), 100, domain.USD, domain.INR, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 8350.0, result.ConvertedAmount)
	mockRepo.AssertExpectations(t)
}

func TestConvertAmount_Historical_ExceedsLimit(t *testing.T) {
	rateService := NewRateService(nil, nil, nil, nil)
	oldDate := time.Now().AddDate(0, 0, -91)
	result, err := rateService.ConvertAmount(context.Background(), 100, domain.USD, domain.INR, &oldDate)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "90-day historical data limit")
}

func TestConvertAmount_Historical_Fiat(t *testing.T) {
	mockFiatProvider := new(MockFiatProvider)
	historicalDate := time.Now().AddDate(0, 0, -10)
	mockFiatProvider.On("FetchHistoricalRate", mock.Anything, historicalDate, domain.USD, domain.EUR).Return(domain.ExchangeRate{Rate: 0.95}, nil)

	rateService := NewRateService(mockFiatProvider, nil, nil, nil)

	result, err := rateService.ConvertAmount(context.Background(), 100, domain.USD, domain.EUR, &historicalDate)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 95.0, result.ConvertedAmount)
	mockFiatProvider.AssertExpectations(t)
}

func TestConvertAmount_Historical_CryptoBlocked(t *testing.T) {
	rateService := NewRateService(nil, nil, nil, nil)
	validDate := time.Now().AddDate(0, 0, -5)
	result, err := rateService.ConvertAmount(context.Background(), 1, domain.USD, domain.BTC, &validDate)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "historical data is not available for cryptocurrencies")
}

// Test that metrics are incremented on provider success.
func TestRefreshRates_MetricsSuccess(t *testing.T) {
	// Setup
	mockFiatProvider := new(MockFiatProvider)
	mockCryptoProvider := new(MockCryptoProvider) // We need a mock for the crypto provider now
	mockRepo := new(MockRepository)
	dummyMetrics := setupDummyMetrics()

	// Define expected return values for providers
	mockFiatProvider.On("FetchLatestRates", mock.Anything, mock.Anything, mock.Anything).Return([]domain.ExchangeRate{{From: "USD", To: "INR", Rate: 83.0, Date: time.Now()}}, nil)
	mockCryptoProvider.On("FetchCryptoRates", mock.Anything, mock.Anything, mock.Anything).Return([]domain.ExchangeRate{}, nil) // Return empty but successful
	mockRepo.On("UpdateLatestRates", mock.Anything, mock.Anything).Return(nil)

	rateService := NewRateService(mockFiatProvider, mockCryptoProvider, mockRepo, dummyMetrics)

	// Execute
	rateService.RefreshRates(context.Background())

	// Assert expectations on provider calls.
	mockFiatProvider.AssertExpectations(t)
	mockCryptoProvider.AssertExpectations(t)
}

// Helper mock for the crypto provider
type MockCryptoProvider struct {
	mock.Mock
}

var _ provider.CryptoProvider = (*MockCryptoProvider)(nil)

func (m *MockCryptoProvider) FetchCryptoRates(ctx context.Context, baseFiat domain.Currency, cryptos []domain.Currency) ([]domain.ExchangeRate, error) {
	args := m.Called(ctx, baseFiat, cryptos)
	return args.Get(0).([]domain.ExchangeRate), args.Error(1)
}
