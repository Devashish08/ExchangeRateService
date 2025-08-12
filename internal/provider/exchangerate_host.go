// Package provider contains clients for third-party exchange-rate APIs.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Devahish08/ExchangeRateService/internal/domain"
)

const exchangeRateHostAPI = "https://api.exchangerate.host/latest"

// ExchangeRateHostProvider implements a client for the exchangerate.host API.
// It is safe for concurrent use.
type ExchangeRateHostProvider struct {
	client *http.Client
}

// NewExchangeRateHostProvider returns a provider with a sensible HTTP client
// timeout for calling the upstream API.
func NewExchangeRateHostProvider() *ExchangeRateHostProvider {
	return &ExchangeRateHostProvider{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// apiResponse models the subset of the upstream JSON response we care about.
type apiResponse struct {
	Rates map[string]float64 `json:"rates"`
	Date  string             `json:"date"`
}

// FetchLatestRates returns the latest rates for the given target currencies
// relative to the provided base currency. The request respects context
// cancellation and deadlines. A non-200 HTTP response or malformed JSON will
// be returned as an error.
func (p *ExchangeRateHostProvider) FetchLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) ([]domain.ExchangeRate, error) {
	var symbols []string
	for _, t := range targets {
		symbols = append(symbols, string(t))
	}

	reqURL := fmt.Sprintf("%s?base=%s&symbols=%s", exchangeRateHostAPI, base, strings.Join(symbols, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request to %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal json response: %w", err)
	}

	date, err := time.Parse("2006-01-02", apiResp.Date)
	if err != nil {
		return nil, fmt.Errorf("parse response date: %w", err)
	}

	var rates []domain.ExchangeRate
	for symbol, rate := range apiResp.Rates {
		rates = append(rates, domain.ExchangeRate{
			From: base,
			To:   domain.Currency(symbol),
			Rate: rate,
			Date: date,
		})
	}

	return rates, nil
}
