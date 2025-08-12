
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
)

const exchangeRateHostBaseURL = "https://api.exchangerate.host"

type ExchangeRateHostProvider struct {
	client *http.Client
	apiKey string
}

func NewExchangeRateHostProvider(apiKey string) *ExchangeRateHostProvider {
	return &ExchangeRateHostProvider{
		client: &http.Client{Timeout: 10 * time.Second},
		apiKey: apiKey,
	}
}

type apiResponse struct {
	Success bool               `json:"success"`
	Source  string             `json:"source"`
	Quotes  map[string]float64 `json:"quotes"` // e.g., "USDAUD": 1.27
	Date    string             `json:"date"`
	Error   struct {
		Info string `json:"info"`
	} `json:"error"`
}

func (p *ExchangeRateHostProvider) FetchLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) ([]domain.ExchangeRate, error) {
	var symbols []string
	for _, t := range targets {
		symbols = append(symbols, string(t))
	}
	reqURL := fmt.Sprintf("%s/live?access_key=%s&source=%s&currencies=%s",
		exchangeRateHostBaseURL, p.apiKey, base, strings.Join(symbols, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request to %s: %w", reqURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d - body: %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal json response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("exchangerate.host API indicated failure: %s", apiResp.Error.Info)
	}

	date, err := time.Parse("2006-01-02", apiResp.Date)
	if err != nil {
		if t, perr := time.Parse(time.RFC3339, apiResp.Date); perr == nil {
			date = t
		} else {
			date = time.Now()
		}
	}

	var rates []domain.ExchangeRate
	for key, rate := range apiResp.Quotes {
		if len(key) == 6 {
			from := domain.Currency(key[0:3])
			to := domain.Currency(key[3:6])
			rates = append(rates, domain.ExchangeRate{
				From: from, To: to, Rate: rate, Date: date,
			})
		}
	}
	return rates, nil
}

func (p *ExchangeRateHostProvider) FetchHistoricalRate(ctx context.Context, date time.Time, base, target domain.Currency) (domain.ExchangeRate, error) {
	dateStr := date.Format("2006-01-02")
	reqURL := fmt.Sprintf("%s/historical?access_key=%s&date=%s&source=%s&currencies=%s",
		exchangeRateHostBaseURL, p.apiKey, dateStr, base, target)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return domain.ExchangeRate{}, fmt.Errorf("create historical request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return domain.ExchangeRate{}, fmt.Errorf("execute historical request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.ExchangeRate{}, fmt.Errorf("read historical response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return domain.ExchangeRate{}, fmt.Errorf("historical API returned non-200 status code: %d - body: %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return domain.ExchangeRate{}, fmt.Errorf("unmarshal historical json response: %w", err)
	}

	if !apiResp.Success {
		return domain.ExchangeRate{}, fmt.Errorf("exchangerate.host API indicated failure for historical rate: %s", apiResp.Error.Info)
	}

	quoteKey := string(base) + string(target)
	rateValue, ok := apiResp.Quotes[quoteKey]
	if !ok {
		return domain.ExchangeRate{}, fmt.Errorf("target currency pair %s not found in historical response", quoteKey)
	}

	return domain.ExchangeRate{
		From: base, To: target, Rate: rateValue, Date: date,
	}, nil
}
