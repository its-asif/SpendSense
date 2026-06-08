package expense

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"spendsense-backend/internal/currency"
	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/wallet"

	"github.com/google/uuid"
)

type fakeExpenseStore struct {
	created        *Expense
	updated        *Expense
	deletedID      uuid.UUID
	deletedUID     uuid.UUID
	getVal         *Expense
	listFrom       *time.Time
	listTo         *time.Time
	listCategoryID *uuid.UUID
}

func (f *fakeExpenseStore) CreateExpense(ctx context.Context, e *Expense) error {
	f.created = e
	return nil
}

func (f *fakeExpenseStore) GetExpenseByID(ctx context.Context, userID, expenseID uuid.UUID) (*Expense, error) {
	if f.getVal != nil {
		return f.getVal, nil
	}
	return nil, domain.NewDomainError(domain.ErrNotFound, "expense not found", 404)
}

func (f *fakeExpenseStore) ListExpenses(ctx context.Context, userID uuid.UUID, limit int, pagination *Pagination, from, to *time.Time, categoryID *uuid.UUID) ([]*Expense, *Pagination, error) {
	f.listFrom = from
	f.listTo = to
	f.listCategoryID = categoryID
	return nil, nil, nil
}

func (f *fakeExpenseStore) UpdateExpense(ctx context.Context, expense *Expense) error {
	f.updated = expense
	return nil
}

func (f *fakeExpenseStore) SoftDeleteExpense(ctx context.Context, userID, expenseID uuid.UUID) error {
	f.deletedUID = userID
	f.deletedID = expenseID
	return nil
}

func TestCreateExpenseValidation(t *testing.T) {
	repo := &fakeExpenseStore{}
	svc := NewService(repo, nil, nil, nil)
	uid := uuid.New()

	// 1. Missing wallet ID
	_, err := svc.CreateExpense(context.Background(), uid, CreateRequest{WalletID: uuid.Nil, CategoryID: uuid.New(), Amount: 10, Date: time.Now()})
	if err == nil {
		t.Fatalf("expected error for missing wallet")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidWallet {
		t.Fatalf("expected ErrInvalidWallet, got %v", err)
	}

	// 2. Missing category ID
	_, err = svc.CreateExpense(context.Background(), uid, CreateRequest{WalletID: uuid.New(), CategoryID: uuid.Nil, Amount: 10, Date: time.Now()})
	if err == nil {
		t.Fatalf("expected error for missing category")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidCategory {
		t.Fatalf("expected ErrInvalidCategory, got %v", err)
	}

	// 3. Invalid amount
	_, err = svc.CreateExpense(context.Background(), uid, CreateRequest{WalletID: uuid.New(), CategoryID: uuid.New(), Amount: 0, Date: time.Now()})
	if err == nil {
		t.Fatalf("expected error for invalid amount")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}

	// 4. Future date
	future := time.Now().AddDate(0, 0, 1)
	_, err = svc.CreateExpense(context.Background(), uid, CreateRequest{WalletID: uuid.New(), CategoryID: uuid.New(), Amount: 10, Date: future})
	if err == nil {
		t.Fatalf("expected error for future date")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidDate {
		t.Fatalf("expected ErrInvalidDate, got %v", err)
	}

	// 5. Invalid recurring rule
	rule := "invalid_rule"
	_, err = svc.CreateExpense(context.Background(), uid, CreateRequest{
		WalletID:      uuid.New(),
		CategoryID:    uuid.New(),
		Amount:        10,
		Date:          time.Now(),
		IsRecurring:   true,
		RecurringRule: &rule,
	})
	if err == nil {
		t.Fatalf("expected error for invalid recurring rule")
	}

	// 6. Valid creation
	now := time.Now()
	expense, err := svc.CreateExpense(context.Background(), uid, CreateRequest{WalletID: uuid.New(), CategoryID: uuid.New(), Amount: 10, Date: now})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expense == nil {
		t.Fatalf("expected expense returned")
	}
	if repo.created == nil {
		t.Fatalf("expected repo.CreateExpense called")
	}
}

func TestListExpensesPassesFilters(t *testing.T) {
	repo := &fakeExpenseStore{}
	svc := NewService(repo, nil, nil, nil)
	uid := uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	categoryID := uuid.New()

	_, _, err := svc.ListExpenses(context.Background(), uid, 20, "", &from, &to, &categoryID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.listFrom == nil || !repo.listFrom.Equal(from) {
		t.Fatalf("expected from filter to be forwarded")
	}
	if repo.listTo == nil || !repo.listTo.Equal(to) {
		t.Fatalf("expected to filter to be forwarded")
	}
	if repo.listCategoryID == nil || *repo.listCategoryID != categoryID {
		t.Fatalf("expected category filter to be forwarded")
	}
}

func TestGetExpense(t *testing.T) {
	userID := uuid.New()
	expenseID := uuid.New()
	expected := &Expense{ID: expenseID, UserID: userID, Amount: 15.5}

	repo := &fakeExpenseStore{getVal: expected}
	svc := NewService(repo, nil, nil, nil)

	// Valid retrieval
	retrieved, err := svc.GetExpense(context.Background(), userID, expenseID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.Amount != 15.5 {
		t.Errorf("expected amount 15.5, got %f", retrieved.Amount)
	}

	// Invalid parameters
	_, err = svc.GetExpense(context.Background(), uuid.Nil, expenseID)
	if err == nil {
		t.Fatalf("expected error for nil user ID")
	}
}

func TestUpdateExpense(t *testing.T) {
	userID := uuid.New()
	expenseID := uuid.New()
	existing := &Expense{ID: expenseID, UserID: userID, Amount: 15.5, CategoryID: uuid.New(), WalletID: uuid.New()}

	repo := &fakeExpenseStore{getVal: existing}
	svc := NewService(repo, nil, nil, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	}

	// Successful update
	updated, err := svc.UpdateExpense(context.Background(), userID, expenseID, UpdateRequest{
		WalletID:   uuid.New(),
		CategoryID: uuid.New(),
		Amount:     22.9,
		Date:       svc.now(),
	})
	if err != nil {
		t.Fatalf("unexpected error updating expense: %v", err)
	}
	if updated.Amount != 22.9 {
		t.Errorf("expected updated amount 22.9, got %f", updated.Amount)
	}
}

func TestSoftDeleteExpense(t *testing.T) {
	userID := uuid.New()
	expenseID := uuid.New()
	repo := &fakeExpenseStore{}
	svc := NewService(repo, nil, nil, nil)

	err := svc.SoftDeleteExpense(context.Background(), userID, expenseID)
	if err != nil {
		t.Fatalf("unexpected error deleting expense: %v", err)
	}
	if repo.deletedID != expenseID || repo.deletedUID != userID {
		t.Errorf("expected repo.SoftDeleteExpense called with %v and %v", userID, expenseID)
	}
}

func TestExpensePagination(t *testing.T) {
	emptyP := Pagination{}
	if encoded := emptyP.Encode(); encoded != "" {
		t.Errorf("expected empty string for zero/nil fields")
	}

	dec, err := DecodePagination("")
	if err != nil || dec != nil {
		t.Errorf("expected nil for empty pagination string")
	}

	_, err = DecodePagination("invalid-base64!")
	if err == nil {
		t.Errorf("expected error decoding invalid base64")
	}

	id := uuid.New()
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	p := Pagination{CreatedAt: now, ID: id}
	encoded := p.Encode()
	decoded, err := DecodePagination(encoded)
	if err != nil {
		t.Fatalf("unexpected error decoding: %v", err)
	}
	if decoded.ID != id || !decoded.CreatedAt.Equal(now) {
		t.Errorf("decoded does not match original pagination struct")
	}
}

type fakeCategoryGuard struct {
	err error
}

func (f *fakeCategoryGuard) AssertAccessible(ctx context.Context, userID, categoryID uuid.UUID, kind string) error {
	return f.err
}

type fakeWalletLookup struct {
	wallet *wallet.Wallet
	err    error
}

func (f *fakeWalletLookup) GetWalletByID(ctx context.Context, userID, id uuid.UUID) (*wallet.Wallet, error) {
	return f.wallet, f.err
}

func TestServiceAdditionalPaths(t *testing.T) {
	t.Run("CreateExpenseErrorPaths", func(t *testing.T) {
		repo := &fakeExpenseStore{}
		cg := &fakeCategoryGuard{err: domain.NewDomainError(domain.ErrForbidden, "forbidden", 403)}
		svc := NewService(repo, nil, cg, nil)

		// 1. Nil user ID
		_, err := svc.CreateExpense(context.Background(), uuid.Nil, CreateRequest{})
		if err == nil {
			t.Errorf("expected error for nil user ID")
		}

		// 2. Category forbidden
		_, err = svc.CreateExpense(context.Background(), uuid.New(), CreateRequest{
			WalletID:   uuid.New(),
			CategoryID: uuid.New(),
			Amount:     10.0,
			Date:       time.Now(),
		})
		if err == nil {
			t.Errorf("expected forbidden category error")
		}
	})

	t.Run("GetExpenseErrorPaths", func(t *testing.T) {
		repo := &fakeExpenseStore{}
		svc := NewService(repo, nil, nil, nil)

		_, err := svc.GetExpense(context.Background(), uuid.Nil, uuid.New())
		if err == nil {
			t.Errorf("expected error for nil userID")
		}

		_, err = svc.GetExpense(context.Background(), uuid.New(), uuid.Nil)
		if err == nil {
			t.Errorf("expected error for nil expenseID")
		}
	})

	t.Run("ListExpensesErrorPaths", func(t *testing.T) {
		repo := &fakeExpenseStore{}
		svc := NewService(repo, nil, nil, nil)

		_, _, err := svc.ListExpenses(context.Background(), uuid.Nil, 10, "", nil, nil, nil)
		if err == nil {
			t.Errorf("expected error for nil userID")
		}

		// Invalid pagination string
		_, _, err = svc.ListExpenses(context.Background(), uuid.New(), -5, "invalid-pagi!", nil, nil, nil)
		if err == nil {
			t.Errorf("expected error for invalid pagination string")
		}
	})

	t.Run("UpdateExpenseErrorPaths", func(t *testing.T) {
		repo := &fakeExpenseStore{}
		cg := &fakeCategoryGuard{err: domain.NewDomainError(domain.ErrForbidden, "forbidden", 403)}
		svc := NewService(repo, nil, cg, nil)

		_, err := svc.UpdateExpense(context.Background(), uuid.Nil, uuid.New(), UpdateRequest{})
		if err == nil {
			t.Errorf("expected error for nil userID")
		}

		_, err = svc.UpdateExpense(context.Background(), uuid.New(), uuid.New(), UpdateRequest{
			WalletID:   uuid.New(),
			CategoryID: uuid.New(),
			Amount:     20.0,
			Date:       time.Now(),
		})
		if err == nil {
			t.Errorf("expected category forbidden error on update")
		}
	})

	t.Run("SoftDeleteExpenseErrorPaths", func(t *testing.T) {
		repo := &fakeExpenseStore{}
		svc := NewService(repo, nil, nil, nil)

		err := svc.SoftDeleteExpense(context.Background(), uuid.Nil, uuid.New())
		if err == nil {
			t.Errorf("expected error for nil userID")
		}
	})
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestServiceCurrencyNormalization(t *testing.T) {
	repo := &fakeExpenseStore{}

	curSvc, err := currency.NewService(nil)
	if err != nil {
		t.Fatalf("unexpected error creating currency service: %v", err)
	}

	curSvc.SetHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"usd": {"eur": 0.85}}`)),
				Header:     make(http.Header),
			}, nil
		}),
	})

	walletObj := &wallet.Wallet{
		ID:         uuid.New(),
		Currency:   "EUR",
		WalletType: "CASH",
	}

	wl := &fakeWalletLookup{wallet: walletObj}
	svc := NewService(repo, wl, nil, curSvc)

	t.Run("ConvertSuccessful", func(t *testing.T) {
		normalized := &normalizedExpenseFields{
			WalletID: walletObj.ID,
			Amount:   100.0,
			Currency: "USD",
		}

		err := svc.normalizeCurrency(context.Background(), uuid.New(), normalized)
		if err != nil {
			t.Fatalf("unexpected error normalizing: %v", err)
		}

		if normalized.Amount != 85.0 || normalized.FXRateToBase != 0.85 || normalized.Currency != "EUR" {
			t.Errorf("unexpected conversion: %+v", normalized)
		}
	})

	t.Run("WalletNotFound", func(t *testing.T) {
		wlErr := &fakeWalletLookup{wallet: nil}
		svcErr := NewService(repo, wlErr, nil, curSvc)
		normalized := &normalizedExpenseFields{
			WalletID: uuid.New(),
			Amount:   100.0,
			Currency: "USD",
		}
		err := svcErr.normalizeCurrency(context.Background(), uuid.New(), normalized)
		if err == nil {
			t.Errorf("expected error for missing wallet")
		}
	})
}

func TestExpenseNormalizationPaths(t *testing.T) {
	svc := NewService(&fakeExpenseStore{}, nil, nil, nil)
	svc.now = func() time.Time { return time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC) }

	uid := uuid.New()

	t.Run("ZeroDate", func(t *testing.T) {
		_, err := svc.CreateExpense(context.Background(), uid, CreateRequest{
			WalletID:   uuid.New(),
			CategoryID: uuid.New(),
			Amount:     10,
			// Date is zero
		})
		if err == nil {
			t.Errorf("expected error for zero date")
		}
	})

	t.Run("NotesEmptyString", func(t *testing.T) {
		empty := "   "
		e, err := svc.CreateExpense(context.Background(), uid, CreateRequest{
			WalletID:   uuid.New(),
			CategoryID: uuid.New(),
			Amount:     10,
			Date:       time.Now(),
			Notes:      &empty,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.Notes != nil {
			t.Errorf("expected nil notes after trimming empty string")
		}
	})

	t.Run("MerchantEmptyString", func(t *testing.T) {
		empty := "   "
		e, err := svc.CreateExpense(context.Background(), uid, CreateRequest{
			WalletID:   uuid.New(),
			CategoryID: uuid.New(),
			Amount:     10,
			Date:       time.Now(),
			Merchant:   &empty,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.Merchant != nil {
			t.Errorf("expected nil merchant after trimming empty string")
		}
	})

	t.Run("DefaultCurrencyUSD", func(t *testing.T) {
		e, err := svc.CreateExpense(context.Background(), uid, CreateRequest{
			WalletID:   uuid.New(),
			CategoryID: uuid.New(),
			Amount:     10,
			Date:       time.Now(),
			// No Currency
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.Currency != "USD" {
			t.Errorf("expected default USD, got %s", e.Currency)
		}
	})

	t.Run("InvalidCurrencyLength", func(t *testing.T) {
		_, err := svc.CreateExpense(context.Background(), uid, CreateRequest{
			WalletID:   uuid.New(),
			CategoryID: uuid.New(),
			Amount:     10,
			Date:       time.Now(),
			Currency:   "USDD",
		})
		if err == nil {
			t.Errorf("expected error for 4-letter currency code")
		}
	})

	t.Run("ValidRecurringRule", func(t *testing.T) {
		for _, rule := range []string{"daily", "weekly", "monthly", "yearly"} {
			r := rule
			e, err := svc.CreateExpense(context.Background(), uid, CreateRequest{
				WalletID:      uuid.New(),
				CategoryID:    uuid.New(),
				Amount:        10,
				Date:          time.Now(),
				IsRecurring:   true,
				RecurringRule: &r,
			})
			if err != nil {
				t.Fatalf("unexpected error for rule %s: %v", rule, err)
			}
			if e.RecurringRule == nil || *e.RecurringRule != rule {
				t.Errorf("expected rule %s, got %v", rule, e.RecurringRule)
			}
		}
	})

	t.Run("EmptyRecurringRule", func(t *testing.T) {
		empty := "   "
		e, err := svc.CreateExpense(context.Background(), uid, CreateRequest{
			WalletID:      uuid.New(),
			CategoryID:    uuid.New(),
			Amount:        10,
			Date:          time.Now(),
			IsRecurring:   true,
			RecurringRule: &empty,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.RecurringRule != nil {
			t.Errorf("expected nil recurring rule after trimming empty string")
		}
	})

	t.Run("UpdateExpenseNilIDs", func(t *testing.T) {
		_, err := svc.UpdateExpense(context.Background(), uuid.Nil, uuid.Nil, UpdateRequest{})
		if err == nil {
			t.Errorf("expected error for nil IDs")
		}
	})

	t.Run("ListExpensesLimitClamping", func(t *testing.T) {
		// limit=0 → 20; limit=200 → 100
		for _, limit := range []int{0, -1, 200} {
			_, _, err := svc.ListExpenses(context.Background(), uid, limit, "", nil, nil, nil)
			if err != nil {
				t.Fatalf("unexpected error for limit %d: %v", limit, err)
			}
		}
	})
}
