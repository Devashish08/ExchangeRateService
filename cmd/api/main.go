// Command server runs the exchange-rate HTTP API and metrics endpoint.

package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiKey := os.Getenv("EXCHANGERATE_API_KEY")
	if apiKey == "" {
		logger.Error("FATAL: EXCHANGERATE_API_KEY environment variable not set.")
		os.Exit(1)
	}

	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics(reg)
	repo := repository.NewInMemoryRateRepository()
	fiatProvider := provider.NewExchangeRateHostProvider(apiKey)
	cryptoProvider := provider.NewCoinGeckoProvider()
	rateService := service.NewRateService(fiatProvider, cryptoProvider, repo, m)

	go func() {
		logger.Info("Performing initial rate refresh...")
		rateService.RefreshRates(context.Background())
	}()

	go startRateRefresher(ctx, logger, rateService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(api.MetricsMiddleware(m))

	r.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	r.Get("/convert", api.NewConversionHandler(rateService).ServeHTTP)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		logger.Info("Starting server", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server startup failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Server shutdown complete.")
}

func startRateRefresher(ctx context.Context, logger *slog.Logger, service *service.RateService) {
	logger.Info("Rate refresher started.")
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Info("Running hourly rate refresh...")
			service.RefreshRates(ctx)
		case <-ctx.Done():
			logger.Info("Rate refresher stopping.")
			return
		}
	}
}
