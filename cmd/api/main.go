package main

import (
	"context"
	"log"
	"time"

	"github.com/Devahish08/ExchangeRateService/internal/provider"
	"github.com/Devahish08/ExchangeRateService/internal/repository"
	"github.com/Devahish08/ExchangeRateService/internal/service"
)

// main wires the application components and performs a one-time rate refresh.
// In production, this would run an HTTP API and schedule periodic refreshes.
func main() {
	prov := provider.NewExchangeRateHostProvider()
	repo := repository.NewInMemoryRateRepository()
	svc := service.NewRateService(prov, repo)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	svc.RefreshRates(ctx)
	log.Println("Startup refresh completed")
}
