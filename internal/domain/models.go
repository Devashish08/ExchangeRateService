package domain

import "time"

// Currency represents a supported fiat or cryptocurrency code.
type Currency string

const (
	USD  Currency = "USD"
	INR  Currency = "INR"
	EUR  Currency = "EUR"
	GBP  Currency = "GBP"
	JPY  Currency = "JPY"
	BTC  Currency = "BTC"
	ETH  Currency = "ETH"
	USDT Currency = "USDT"
)

// IsSupported reports whether the currency is supported by the service.
func (c Currency) IsSupported() bool {
	switch c {
	case USD, INR, EUR, JPY, GBP, BTC, ETH, USDT:
		return true
	default:
		return false
	}
}

// IsCrypto reports whether the currency is a cryptocurrency.
func (c Currency) IsCrypto() bool {
	switch c {
	case BTC, ETH, USDT:
		return true
	default:
		return false
	}
}

// ExchangeRate captures a conversion rate from one currency to another at a given time.
type ExchangeRate struct {
	From Currency
	To   Currency
	Rate float64
	Date time.Time
}

// ConversionResult is the API response payload for a conversion request.
type ConversionResult struct {
	ConvertedAmount float64 `json:"amount"`
}
