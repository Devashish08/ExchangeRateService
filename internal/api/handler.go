package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Devashish08/ExchangeRateService/internal/domain"
	"github.com/Devashish08/ExchangeRateService/internal/service"

	"github.com/go-chi/render"
)

// ConversionHandler handles currency conversion HTTP requests.
type ConversionHandler struct {
	rateService *service.RateService
}

// NewConversionHandler constructs a ConversionHandler bound to a RateService.
func NewConversionHandler(s *service.RateService) *ConversionHandler {
	return &ConversionHandler{rateService: s}
}

// ServeHTTP handles requests to the /convert endpoint.
func (h *ConversionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fromParam := r.URL.Query().Get("from")
	toParam := r.URL.Query().Get("to")
	amountParam := r.URL.Query().Get("amount")
	dateParam := r.URL.Query().Get("date")

	if fromParam == "" || toParam == "" || amountParam == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "missing required query parameters: from, to, amount"})
		return
	}

	fromCurrency := domain.Currency(fromParam)
	toCurrency := domain.Currency(toParam)
	if !fromCurrency.IsSupported() || !toCurrency.IsSupported() {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "one or more currencies are not supported"})
		return
	}

	amount, err := strconv.ParseFloat(amountParam, 64)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{"error": "invalid amount format"})
		return
	}

	var date *time.Time
	if dateParam != "" {
		parsedDate, err := time.Parse("2006-01-02", dateParam)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": "invalid date format, please use YYYY-MM-DD"})
			return
		}
		date = &parsedDate
	}

	result, err := h.rateService.ConvertAmount(r.Context(), amount, fromCurrency, toCurrency, date)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}
	render.JSON(w, r, result)
}
