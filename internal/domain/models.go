package domain

import "time"

type Currency string

const (
	USD Currency = "USD"
	INR Currency = "INR"
	EUR Currency = "EUR"
	GBP Currency = "GBP"
	JPY Currency = "JPY"
)

func (c Currency) IsSupported() bool {
	switch c {
	case USD, INR, EUR, GBP, JPY:
		return true
	default:
		return false
	}
}

type ExchangeRate struct {
	From Currency
	To   Currency
	Rate float64
	Date time.Time
}

type ConversionResult struct {
	ConvertedAmount float64 `json:"amount"`
}
