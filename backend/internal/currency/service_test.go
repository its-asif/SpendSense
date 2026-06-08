package currency

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type noopCache struct{}

func (n noopCache) Get(ctx context.Context, key string) (string, error)                 { return "", context.Canceled }
func (n noopCache) Set(ctx context.Context, key, value string, ttl time.Duration) error { return nil }

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestListCurrenciesPlacesDefaultFirst(t *testing.T) {
	svc, err := NewService(nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	items := svc.ListCurrencies("BDT")
	if len(items) == 0 {
		t.Fatalf("expected currencies")
	}
	if items[0].Code != "BDT" || !items[0].IsDefault {
		t.Fatalf("expected BDT to be first default, got %+v", items[0])
	}
}

func TestConvertDetailedSameCurrency(t *testing.T) {
	svc, err := NewService(noopCache{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := svc.ConvertDetailed(context.Background(), 123.45, "USD", "USD")
	if err != nil {
		t.Fatalf("convert detailed: %v", err)
	}
	if result.ExchangeRate != 1 || result.ConvertedAmount != 123.45 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestMetadata(t *testing.T) {
	svc, err := NewService(nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	// 1. Valid currency metadata
	meta, found := svc.Metadata("USD")
	if !found {
		t.Errorf("expected USD metadata to be found")
	}
	if meta.Code != "USD" || meta.Symbol != "$" {
		t.Errorf("incorrect USD metadata: %+v", meta)
	}

	// 2. Invalid currency metadata
	_, found = svc.Metadata("XYZ")
	if found {
		t.Errorf("expected XYZ metadata not to be found")
	}
}

func TestDefaultCurrencyForWallet(t *testing.T) {
	svc, err := NewService(nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	// 1. Provider priority defaults
	bkash := "bkash"
	if cur := svc.DefaultCurrencyForWallet("CASH", &bkash); cur != "BDT" {
		t.Errorf("expected BDT for bkash, got %s", cur)
	}

	amex := "AMEX"
	if cur := svc.DefaultCurrencyForWallet("CASH", &amex); cur != "USD" {
		t.Errorf("expected USD for amex, got %s", cur)
	}

	// 2. Wallet type fallback defaults
	if cur := svc.DefaultCurrencyForWallet("cash", nil); cur != "BDT" {
		t.Errorf("expected BDT for cash type, got %s", cur)
	}
	if cur := svc.DefaultCurrencyForWallet("card", nil); cur != "USD" {
		t.Errorf("expected USD for card type, got %s", cur)
	}
	if cur := svc.DefaultCurrencyForWallet("unknown", nil); cur != "USD" {
		t.Errorf("expected USD default fallback, got %s", cur)
	}
}

func TestNormalizeOrDefault(t *testing.T) {
	svc, err := NewService(nil)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	// 1. Valid code
	norm, err := svc.NormalizeOrDefault(" bdt ", "CASH", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if norm != "BDT" {
		t.Errorf("expected BDT, got %s", norm)
	}

	// 2. Empty code should fallback to default
	norm, err = svc.NormalizeOrDefault("", "cash", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if norm != "BDT" {
		t.Errorf("expected BDT for empty code + cash wallet, got %s", norm)
	}

	// 3. Invalid code should error
	_, err = svc.NormalizeOrDefault("XYZ", "CASH", nil)
	if err == nil {
		t.Fatalf("expected error for unsupported currency XYZ")
	}
}

func TestExchangeRateAndConvertDetailed(t *testing.T) {
	svc, err := NewService(noopCache{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	// Mock the HTTP transport
	svc.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"usd": {"eur": 0.85}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	rate, err := svc.ExchangeRate(context.Background(), "USD", "EUR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate != 0.85 {
		t.Errorf("expected 0.85 exchange rate, got %f", rate)
	}

	// Test ConvertDetailed
	result, err := svc.ConvertDetailed(context.Background(), 200.0, "USD", "EUR")
	if err != nil {
		t.Fatalf("unexpected convert detailed error: %v", err)
	}
	if result.ConvertedAmount != 170.0 || result.ExchangeRate != 0.85 {
		t.Errorf("expected converted amount 170.0 and rate 0.85, got %+v", result)
	}
}
