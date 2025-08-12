// Command server runs the exchange-rate HTTP API and metrics endpoint.

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/api"
	"github.com/Devashish08/ExchangeRateService/internal/metrics"
	"github.com/Devashish08/ExchangeRateService/internal/provider"
	"github.com/Devashish08/ExchangeRateService/internal/repository"
	"github.com/Devashish08/ExchangeRateService/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics(reg)

	apiKey := os.Getenv("EXCHANGERATE_API_KEY")
	if apiKey == "" {
		log.Fatal("FATAL: EXCHANGERATE_API_KEY environment variable not set.")
	}

	ctx := context.Background()
	repo := repository.NewInMemoryRateRepository()
	fiatProvider := provider.NewExchangeRateHostProvider(apiKey)
	cryptoProvider := provider.NewCoinGeckoProvider()
	rateService := service.NewRateService(fiatProvider, cryptoProvider, repo, m)


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
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(api.MetricsMiddleware(m))

	r.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	r.Get("/convert", conversionHandler.ServeHTTP)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
