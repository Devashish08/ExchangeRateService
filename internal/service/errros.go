package service

import "errors"

var (
	ErrDateOutOfRange = errors.New("date is beyond the 90-day historical data limit")
	ErrHistoricalCrypto = errors.New("historical data is not available for cryptocurrencies")
)