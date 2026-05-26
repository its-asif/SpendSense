package currency

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed data/common_currency.json
var commonCurrencyJSON []byte

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

type Metadata struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	SymbolNative  string  `json:"symbol_native"`
	DecimalDigits int     `json:"decimal_digits"`
	Rounding      float64 `json:"rounding"`
	Code          string  `json:"code"`
	NamePlural    string  `json:"name_plural"`
}

type CurrencyOption struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Symbol        string  `json:"symbol"`
	SymbolNative  string  `json:"symbol_native"`
	DecimalDigits int     `json:"decimal_digits"`
	Rounding      float64 `json:"rounding"`
	NamePlural    string  `json:"name_plural"`
	IsDefault     bool    `json:"is_default"`
}

type ConversionResult struct {
	FromCurrency    string  `json:"from_currency"`
	ToCurrency      string  `json:"to_currency"`
	Amount          float64 `json:"amount"`
	ConvertedAmount float64 `json:"converted_amount"`
	ExchangeRate    float64 `json:"exchange_rate"`
}

// exchangeAPIResponse was previously a strict map[string]map[string]float64 but
// some providers may return numeric values as strings. We decode into
// map[string]map[string]interface{} and coerce to float64 below.

type Service struct {
	metadata map[string]Metadata
	cache    Cache
	http     *http.Client
}

func NewService(cache Cache) (*Service, error) {
	metadata := map[string]Metadata{}
	if err := json.Unmarshal(commonCurrencyJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to load currency metadata: %w", err)
	}

	return &Service{
		metadata: metadata,
		cache:    cache,
		http:     &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (s *Service) Metadata(code string) (Metadata, bool) {
	normalized := normalizeCode(code)
	meta, ok := s.metadata[normalized]
	return meta, ok
}

func (s *Service) ListCurrencies(defaultCode string) []CurrencyOption {
	defaultCode = normalizeCode(defaultCode)
	if defaultCode == "" {
		defaultCode = "USD"
	}

	options := make([]CurrencyOption, 0, len(s.metadata))
	for code, meta := range s.metadata {
		options = append(options, CurrencyOption{
			Code:          code,
			Name:          meta.Name,
			Symbol:        meta.Symbol,
			SymbolNative:  meta.SymbolNative,
			DecimalDigits: meta.DecimalDigits,
			Rounding:      meta.Rounding,
			NamePlural:    meta.NamePlural,
			IsDefault:     code == defaultCode,
		})
	}

	sortCurrencies(options)
	return options
}

func (s *Service) NormalizeOrDefault(code, walletType string, provider *string) (string, error) {
	normalized := normalizeCode(code)
	if normalized == "" {
		normalized = s.DefaultCurrencyForWallet(walletType, provider)
	}
	if _, ok := s.Metadata(normalized); !ok {
		return "", fmt.Errorf("unsupported currency: %s", normalized)
	}
	return normalized, nil
}

func (s *Service) DefaultCurrencyForWallet(walletType string, provider *string) string {
	providerKey := strings.ToLower(strings.TrimSpace(pointerString(provider)))
	if providerKey != "" {
		if currency := providerCurrencyDefaults[providerKey]; currency != "" {
			return currency
		}
	}

	walletKey := strings.ToLower(strings.TrimSpace(walletType))
	if currency := walletTypeCurrencyDefaults[walletKey]; currency != "" {
		return currency
	}

	return "USD"
}

func (s *Service) ExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (float64, error) {
	fromCurrency = normalizeCode(fromCurrency)
	toCurrency = normalizeCode(toCurrency)
	if fromCurrency == "" || toCurrency == "" {
		return 0, fmt.Errorf("currency codes are required")
	}
	if fromCurrency == toCurrency {
		return 1, nil
	}

	cacheKey := exchangeCacheKey(fromCurrency, toCurrency)
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
			var rate float64
			if err := json.Unmarshal([]byte(cached), &rate); err == nil && rate > 0 {
				return rate, nil
			}
		}
	}

	url := fmt.Sprintf("https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/%s.json", strings.ToLower(fromCurrency))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := s.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to fetch currency rates: %s", resp.Status)
	}

	// Read the response body so we can attempt multiple decode strategies
	// and include a useful snippet in error messages.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read exchange API response: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		// Some APIs return a JSON string containing the actual JSON object.
		var asString string
		if err2 := json.Unmarshal(body, &asString); err2 == nil {
			// try to unmarshal the inner JSON
			if err3 := json.Unmarshal([]byte(asString), &payload); err3 != nil {
				snippet := string(body)
				if len(snippet) > 512 {
					snippet = snippet[:512]
				}
				return 0, fmt.Errorf("failed to decode exchange API response: %v; inner decode: %v; body: %s", err, err3, snippet)
			}
		} else {
			snippet := string(body)
			if len(snippet) > 512 {
				snippet = snippet[:512]
			}
			return 0, fmt.Errorf("failed to decode exchange API response: %v; body: %s", err, snippet)
		}
	}
	// The API response often includes a top-level "date" and then a
	// map for the source currency (e.g. "usd": { "bdt": 115.0, ... }).
	// Find the map for the fromCurrency key.
	key := strings.ToLower(fromCurrency)
	maybeRates, ok := payload[key]
	if !ok {
		// Sometimes the provider nests rates under a different key; fail clearly.
		snippet := string(body)
		if len(snippet) > 512 {
			snippet = snippet[:512]
		}
		return 0, fmt.Errorf("exchange rate not available for %s to %s; body: %s", fromCurrency, toCurrency, snippet)
	}

	ratesMap, ok := maybeRates.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected rates structure for %s: %T", key, maybeRates)
	}

	raw, ok := ratesMap[strings.ToLower(toCurrency)]
	if !ok {
		return 0, fmt.Errorf("exchange rate not available for %s to %s", fromCurrency, toCurrency)
	}

	// Coerce raw into float64 from common possible types.
	var rate float64
	switch v := raw.(type) {
	case float64:
		rate = v
	case json.Number:
		parsed, err := v.Float64()
		if err != nil {
			return 0, fmt.Errorf("invalid exchange rate number: %w", err)
		}
		rate = parsed
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid exchange rate format: %w", err)
		}
		rate = parsed
	case int:
		rate = float64(v)
	case int64:
		rate = float64(v)
	default:
		return 0, fmt.Errorf("unexpected exchange rate type: %T", v)
	}

	if rate <= 0 {
		return 0, fmt.Errorf("exchange rate not available for %s to %s", fromCurrency, toCurrency)
	}

	rate = round4(rate)

	if s.cache != nil {
		if encoded, err := json.Marshal(rate); err == nil {
			_ = s.cache.Set(ctx, cacheKey, string(encoded), 24*time.Hour)
		}
	}

	return rate, nil
}

func (s *Service) Convert(ctx context.Context, amount float64, fromCurrency, toCurrency string) (float64, float64, error) {
	rate, err := s.ExchangeRate(ctx, fromCurrency, toCurrency)
	if err != nil {
		return 0, 0, err
	}
	return round2(amount * rate), rate, nil
}

func (s *Service) ConvertDetailed(ctx context.Context, amount float64, fromCurrency, toCurrency string) (*ConversionResult, error) {
	convertedAmount, rate, err := s.Convert(ctx, amount, fromCurrency, toCurrency)
	if err != nil {
		return nil, err
	}

	return &ConversionResult{
		FromCurrency:    normalizeCode(fromCurrency),
		ToCurrency:      normalizeCode(toCurrency),
		Amount:          round2(amount),
		ConvertedAmount: convertedAmount,
		ExchangeRate:    rate,
	}, nil
}

func normalizeCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func exchangeCacheKey(fromCurrency, toCurrency string) string {
	return fmt.Sprintf("currency:fx:%s:%s", strings.ToLower(fromCurrency), strings.ToLower(toCurrency))
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func sortCurrencies(items []CurrencyOption) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDefault != items[j].IsDefault {
			return items[i].IsDefault
		}
		return items[i].Code < items[j].Code
	})
}

var providerCurrencyDefaults = map[string]string{
	"bkash":            "BDT",
	"nagad":            "BDT",
	"rocket":           "BDT",
	"american express": "USD",
	"amex":             "USD",
	"visa":             "USD",
	"mastercard":       "USD",
}

var walletTypeCurrencyDefaults = map[string]string{
	"cash":          "BDT",
	"mobile_wallet": "BDT",
	"bank":          "BDT",
	"card":          "USD",
	"default":       "USD",
}
