package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
)

const DefaultExchangeRateHostBaseURL = "https://api.exchangerate.host"

type ExchangeRateHostProvider struct {
	client  *http.Client
	apiKey  string
	baseURL string
}

func NewExchangeRateHostProvider(apiKey, baseURL string) *ExchangeRateHostProvider {
	return &ExchangeRateHostProvider{
		client:  &http.Client{Timeout: 10 * time.Second},
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

type apiResponse struct {
	Success   bool               `json:"success"`
	Source    string             `json:"source"`
	Quotes    map[string]float64 `json:"quotes"`
	Date      string             `json:"date"`
	Timestamp int64              `json:"timestamp"`
	Error     struct {
		Info string `json:"info"`
	} `json:"error"`
}

func (p *ExchangeRateHostProvider) doRequest(ctx context.Context, url string) (*apiResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal json response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("exchangerate.host API indicated failure: %s", apiResp.Error.Info)
	}

	return &apiResp, nil
}

func (p *ExchangeRateHostProvider) FetchLatestRates(ctx context.Context, base domain.Currency, targets []domain.Currency) ([]domain.ExchangeRate, error) {
	var symbols []string
	for _, t := range targets {
		symbols = append(symbols, string(t))
	}
	reqURL := fmt.Sprintf("%s/live?access_key=%s&source=%s&currencies=%s",
		p.baseURL, p.apiKey, base, strings.Join(symbols, ","))

	apiResp, err := p.doRequest(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	if apiResp.Timestamp == 0 {
		return nil, fmt.Errorf("API response missing valid timestamp")
	}
	date := time.Unix(apiResp.Timestamp, 0).UTC()

	var rates []domain.ExchangeRate
	for key, rate := range apiResp.Quotes {
		if len(key) != 6 {
			continue
		}

		from := domain.Currency(key[0:3])
		to := domain.Currency(key[3:6])

		if from.IsSupported() && to.IsSupported() {
			rates = append(rates, domain.ExchangeRate{
				From: from, To: to, Rate: rate, Date: date,
			})
		}
	}
	return rates, nil
}

func (p *ExchangeRateHostProvider) FetchHistoricalRates(ctx context.Context, date time.Time, base domain.Currency, targets []domain.Currency) ([]domain.ExchangeRate, error) {
	dateStr := date.Format("2006-01-02")
	var symbols []string
	for _, t := range targets {
		symbols = append(symbols, string(t))
	}

	reqURL := fmt.Sprintf("%s/historical?access_key=%s&date=%s&source=%s&currencies=%s",
		p.baseURL, p.apiKey, dateStr, base, strings.Join(symbols, ","))

	apiResp, err := p.doRequest(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var rates []domain.ExchangeRate
	for key, rate := range apiResp.Quotes {
		if len(key) != 6 {
			continue
		}

		from := domain.Currency(key[0:3])
		to := domain.Currency(key[3:6])

		if from.IsSupported() && to.IsSupported() {
			rates = append(rates, domain.ExchangeRate{
				From: from, To: to, Rate: rate, Date: date,
			})
		}
	}
	return rates, nil
}
