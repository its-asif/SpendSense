package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

type currencyOptionResponse struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Symbol        string  `json:"symbol"`
	SymbolNative  string  `json:"symbol_native"`
	DecimalDigits int     `json:"decimal_digits"`
	Rounding      float64 `json:"rounding"`
	NamePlural    string  `json:"name_plural"`
	IsDefault     bool    `json:"is_default"`
}

type currencyListResponse struct {
	DefaultCurrency string                   `json:"default_currency"`
	Currencies      []currencyOptionResponse `json:"currencies"`
}

type currencyConvertResponse struct {
	FromCurrency    string  `json:"from_currency"`
	ToCurrency      string  `json:"to_currency"`
	Amount          float64 `json:"amount"`
	ConvertedAmount float64 `json:"converted_amount"`
	ExchangeRate    float64 `json:"exchange_rate"`
}

func (s *Server) handleCurrencyList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	defaultCurrency := strings.TrimSpace(r.URL.Query().Get("default_currency"))
	if defaultCurrency == "" {
		defaultCurrency = "USD"
	}
	options := s.currencyService.ListCurrencies(defaultCurrency)

	resp := make([]currencyOptionResponse, 0, len(options))
	for _, item := range options {
		resp = append(resp, currencyOptionResponse{
			Code:          item.Code,
			Name:          item.Name,
			Symbol:        item.Symbol,
			SymbolNative:  item.SymbolNative,
			DecimalDigits: item.DecimalDigits,
			Rounding:      item.Rounding,
			NamePlural:    item.NamePlural,
			IsDefault:     item.IsDefault,
		})
	}

	writeJSON(w, http.StatusOK, currencyListResponse{
		DefaultCurrency: defaultCurrency,
		Currencies:      resp,
	})
}

func (s *Server) handleCurrencyConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(r.URL.Query().Get("amount")), 64)
	if err != nil || amount <= 0 {
		writeStatusError(w, http.StatusBadRequest, "INVALID_AMOUNT", "amount must be greater than zero")
		return
	}

	fromCurrency := strings.TrimSpace(r.URL.Query().Get("from"))
	if fromCurrency == "" {
		fromCurrency = strings.TrimSpace(r.URL.Query().Get("from_currency"))
	}
	toCurrency := strings.TrimSpace(r.URL.Query().Get("to"))
	if toCurrency == "" {
		toCurrency = strings.TrimSpace(r.URL.Query().Get("to_currency"))
	}
	if fromCurrency == "" || toCurrency == "" {
		writeStatusError(w, http.StatusBadRequest, "INVALID_CURRENCY", "from and to currencies are required")
		return
	}

	result, err := s.currencyService.ConvertDetailed(r.Context(), amount, fromCurrency, toCurrency)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, currencyConvertResponse{
		FromCurrency:    result.FromCurrency,
		ToCurrency:      result.ToCurrency,
		Amount:          result.Amount,
		ConvertedAmount: result.ConvertedAmount,
		ExchangeRate:    result.ExchangeRate,
	})
}
