// In file: internal/provider/coingecko.go

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

const coinGeckoAPI = "https://api.coingecko.com/api/v3/simple/price"

	
var cryptoIDMap = map[domain.Currency]string{
	domain.BTC:  "bitcoin",
	domain.ETH:  "ethereum",
	domain.USDT: "tether",
}

type CoinGeckoProvider struct {
	client *http.Client
}

func NewCoinGeckoProvider() *CoinGeckoProvider {
	return &CoinGeckoProvider{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *CoinGeckoProvider) FetchCryptoRates(ctx context.Context, baseFiat domain.Currency, cryptos []domain.Currency) ([]domain.ExchangeRate, error) {
	var cryptoIDs []string
	
	for _, c := range cryptos {
		if id, ok := cryptoIDMap[c]; ok {
			cryptoIDs = append(cryptoIDs, id)
		}
	}

	if len(cryptoIDs) == 0 {
		return []domain.ExchangeRate{}, nil // Nothing to fetch
	}

	reqURL := fmt.Sprintf("%s?ids=%s&vs_currencies=%s",
		coinGeckoAPI,
		strings.Join(cryptoIDs, ","),
		strings.ToLower(string(baseFiat)),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create coingecko request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute coingecko request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coingecko returned non-200 status code: %d", resp.StatusCode)
	}

	var apiResp map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal coingecko response: %w", err)
	}

	var rates []domain.ExchangeRate
	date := time.Now().UTC()

	reverseIDMap := make(map[string]domain.Currency)
	for symbol, id := range cryptoIDMap {
		reverseIDMap[id] = symbol
	}

	for cryptoID, priceMap := range apiResp {
		rate, ok := priceMap[strings.ToLower(string(baseFiat))]
		if !ok {
			continue
		}
		
		symbol, ok := reverseIDMap[cryptoID]
		if !ok {
			continue
		}
		rates = append(rates, domain.ExchangeRate{
			From: baseFiat,
			To:   symbol,
			Rate: rate,
			Date: date,
		})
	}

	return rates, nil
}
