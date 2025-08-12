package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
	"github.com/Devashish08/ExchangeRateService/internal/repository"
	"github.com/Devashish08/ExchangeRateService/internal/service"

	"github.com/go-chi/render"
)

// ConversionHandler handles currency conversion HTTP requests.
type ConversionHandler struct {
	rateService *service.RateService
	logger      *slog.Logger
}

// NewConversionHandler constructs a ConversionHandler bound to a RateService.
func NewConversionHandler(s *service.RateService, l *slog.Logger) *ConversionHandler {
	return &ConversionHandler{rateService: s, logger: l}
}

// ServeHTTP handles requests to the /convert endpoint.
func (h *ConversionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fromParam := r.URL.Query().Get("from")
	toParam := r.URL.Query().Get("to")
	amountParam := r.URL.Query().Get("amount")
	dateParam := r.URL.Query().Get("date")

	log := h.logger.With("path", r.URL.Path, "method", r.Method)

	if fromParam == "" || toParam == "" || amountParam == "" {
		log.Warn("missing required query parameters", "from", fromParam, "to", toParam, "amount", amountParam)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required query parameters: from, to, amount"})
		return
	}

	fromCurrency := domain.Currency(fromParam)
	toCurrency := domain.Currency(toParam)
	if !fromCurrency.IsSupported() || !toCurrency.IsSupported() {
		log.Warn("unsupported currency provided", "from", fromParam, "to", toParam)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "one or more currencies are not supported"})
		return
	}

	amount, err := strconv.ParseFloat(amountParam, 64)
	if err != nil {
		log.Warn("invalid amount parameter received", "amount", amountParam)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid amount format"})
		return
	}

	var date *time.Time
	if dateParam != "" {
		parsedDate, err := time.Parse("2006-01-02", dateParam)
		if err != nil {
			log.Warn("invalid date parameter received", "date", dateParam)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "invalid date format, please use YYYY-MM-DD"})
			return
		}
		date = &parsedDate
	}

	result, err := h.rateService.ConvertAmount(r.Context(), amount, fromCurrency, toCurrency, date)
	if err != nil {
		if errors.Is(err, service.ErrDateOutOfRange) || errors.Is(err, service.ErrHistoricalCrypto) {
			log.Warn("client validation error during conversion", "error", err.Error())
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": err.Error()})
		} else if errors.Is(err, repository.ErrRateNotFound) {
			log.Warn("rate not found for currency pair", "error", err.Error())
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": err.Error()})
		} else {

			log.Error("unhandled internal error during conversion", "error", err.Error())
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": "an internal server error occurred"})
		}
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, result)
}
