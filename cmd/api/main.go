// In file: cmd/api/main.go

package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/api"
	"github.com/Devashish08/ExchangeRateService/internal/provider"
	"github.com/Devashish08/ExchangeRateService/internal/repository"
	"github.com/Devashish08/ExchangeRateService/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	ctx := context.Background()
	repo := repository.NewInMemoryRateRepository()
	prov := provider.NewExchangeRateHostProvider()
	rateService := service.NewRateService(prov, repo)

	log.Println("Performing initial rate refresh...")
	rateService.RefreshRates(ctx)

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rateService.RefreshRates(context.Background())
			}
		}
	}()

	conversionHandler := api.NewConversionHandler(rateService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)    // Basic logging middleware
	r.Use(middleware.Recoverer) // Panic recovery

	r.Get("/convert", conversionHandler.ServeHTTP)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
