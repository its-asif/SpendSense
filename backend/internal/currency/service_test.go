package currency

import (
	"context"
	"testing"
	"time"
)

type noopCache struct{}

func (n noopCache) Get(ctx context.Context, key string) (string, error)                 { return "", context.Canceled }
func (n noopCache) Set(ctx context.Context, key, value string, ttl time.Duration) error { return nil }

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
