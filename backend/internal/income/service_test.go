package income

import (
	"context"
	"testing"
	"time"

	"spendsense-backend/internal/currency"
	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/wallet"

	"github.com/google/uuid"
)

type fakeIncomeStore struct {
	created    *Income
	updated    *Income
	deletedID  uuid.UUID
	deletedUID uuid.UUID
	getVal     *Income
	listVal    []*Income
	listPage   *Pagination
}

func (f *fakeIncomeStore) CreateIncome(ctx context.Context, income *Income) error {
	f.created = income
	income.CreatedAt = time.Now()
	income.UpdatedAt = time.Now()
	return nil
}

func (f *fakeIncomeStore) GetIncomeByID(ctx context.Context, userID, incomeID uuid.UUID) (*Income, error) {
	if f.getVal != nil {
		return f.getVal, nil
	}
	return nil, domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
}

func (f *fakeIncomeStore) ListIncomes(ctx context.Context, userID uuid.UUID, limit int, pagination *Pagination) ([]*Income, *Pagination, error) {
	return f.listVal, f.listPage, nil
}

func (f *fakeIncomeStore) UpdateIncome(ctx context.Context, income *Income) error {
	f.updated = income
	return nil
}

func (f *fakeIncomeStore) SoftDeleteIncome(ctx context.Context, userID, incomeID uuid.UUID) error {
	f.deletedUID = userID
	f.deletedID = incomeID
	return nil
}

type fakeWalletLookup struct {
	wallet *wallet.Wallet
}

func (fw fakeWalletLookup) GetWalletByID(ctx context.Context, userID, id uuid.UUID) (*wallet.Wallet, error) {
	if fw.wallet != nil {
		return fw.wallet, nil
	}
	return nil, domain.NewDomainError(domain.ErrNotFound, "wallet not found", 404)
}

type fakeCategoryGuard struct {
	err error
}

func (fc fakeCategoryGuard) AssertAccessible(ctx context.Context, userID, categoryID uuid.UUID, kind string) error {
	return fc.err
}

func TestCreateIncomeValidation(t *testing.T) {
	repo := &fakeIncomeStore{}
	wl := fakeWalletLookup{
		wallet: &wallet.Wallet{
			ID:         uuid.New(),
			Currency:   "USD",
			WalletType: "CASH",
		},
	}
	cg := fakeCategoryGuard{}
	svc := NewService(repo, wl, cg, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	}

	userID := uuid.New()

	// 1. Missing user ID
	_, err := svc.CreateIncome(context.Background(), uuid.Nil, CreateRequest{})
	if err == nil {
		t.Fatalf("expected error for empty user ID")
	}

	// 2. Missing wallet ID
	_, err = svc.CreateIncome(context.Background(), userID, CreateRequest{
		SourceName: "Salary",
		Amount:     100,
		Currency:   "USD",
		IncomeDate: svc.now(),
	})
	if err == nil {
		t.Fatalf("expected error for empty wallet ID")
	}

	// 3. Missing source name
	_, err = svc.CreateIncome(context.Background(), userID, CreateRequest{
		WalletID:   uuid.New(),
		Amount:     100,
		Currency:   "USD",
		IncomeDate: svc.now(),
	})
	if err == nil {
		t.Fatalf("expected error for empty source name")
	}

	// 4. Invalid amount
	_, err = svc.CreateIncome(context.Background(), userID, CreateRequest{
		WalletID:   uuid.New(),
		SourceName: "Salary",
		Amount:     -5,
		Currency:   "USD",
		IncomeDate: svc.now(),
	})
	if err == nil {
		t.Fatalf("expected error for negative amount")
	}

	// 5. Future date
	_, err = svc.CreateIncome(context.Background(), userID, CreateRequest{
		WalletID:   uuid.New(),
		SourceName: "Salary",
		Amount:     100,
		Currency:   "USD",
		IncomeDate: svc.now().AddDate(0, 0, 1),
	})
	if err == nil {
		t.Fatalf("expected error for future date")
	}

	// 6. Invalid currency length
	_, err = svc.CreateIncome(context.Background(), userID, CreateRequest{
		WalletID:   uuid.New(),
		SourceName: "Salary",
		Amount:     100,
		Currency:   "USDD",
		IncomeDate: svc.now(),
	})
	if err == nil {
		t.Fatalf("expected error for invalid currency length")
	}

	// 7. Successful creation
	walletID := uuid.New()
	categoryID := uuid.New()
	notes := "bonus"
	income, err := svc.CreateIncome(context.Background(), userID, CreateRequest{
		WalletID:   walletID,
		CategoryID: &categoryID,
		SourceName: "Bonus",
		Amount:     250.556,
		Currency:   "USD",
		IncomeDate: svc.now(),
		Notes:      &notes,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if income == nil {
		t.Fatalf("expected income structure returned")
	}
	if income.Amount != 250.56 {
		t.Errorf("expected rounded amount 250.56, got %f", income.Amount)
	}
	if income.SourceName != "Bonus" {
		t.Errorf("expected SourceName Bonus, got %s", income.SourceName)
	}
}

func TestGetIncome(t *testing.T) {
	userID := uuid.New()
	incomeID := uuid.New()
	expected := &Income{ID: incomeID, UserID: userID, SourceName: "Retirement"}

	repo := &fakeIncomeStore{getVal: expected}
	svc := NewService(repo, nil, nil, nil)

	// Valid retrieval
	retrieved, err := svc.GetIncome(context.Background(), userID, incomeID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.SourceName != "Retirement" {
		t.Errorf("expected Retirement, got %s", retrieved.SourceName)
	}

	// Invalid parameters
	_, err = svc.GetIncome(context.Background(), uuid.Nil, incomeID)
	if err == nil {
		t.Fatalf("expected error for nil user ID")
	}

	_, err = svc.GetIncome(context.Background(), userID, uuid.Nil)
	if err == nil {
		t.Fatalf("expected error for nil income ID")
	}
}

func TestUpdateIncome(t *testing.T) {
	userID := uuid.New()
	incomeID := uuid.New()
	existing := &Income{ID: incomeID, UserID: userID, SourceName: "Gigs", Amount: 50, Currency: "USD"}

	repo := &fakeIncomeStore{getVal: existing}
	svc := NewService(repo, nil, nil, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	}

	// Successful update
	categoryID := uuid.New()
	updated, err := svc.UpdateIncome(context.Background(), userID, incomeID, UpdateRequest{
		WalletID:   uuid.New(),
		CategoryID: &categoryID,
		SourceName: "Freelance",
		Amount:     120,
		Currency:   "USD",
		IncomeDate: svc.now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.SourceName != "Freelance" || updated.Amount != 120 {
		t.Errorf("unexpected update output: %+v", updated)
	}
}

func TestListIncomes(t *testing.T) {
	userID := uuid.New()
	repo := &fakeIncomeStore{
		listVal: []*Income{
			{ID: uuid.New(), UserID: userID, SourceName: "Dividend"},
		},
	}
	svc := NewService(repo, nil, nil, nil)

	incomes, next, err := svc.ListIncomes(context.Background(), userID, 10, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(incomes) != 1 || incomes[0].SourceName != "Dividend" {
		t.Errorf("unexpected list output")
	}
	if next != "" {
		t.Errorf("expected empty next pagination token")
	}
}

func TestSoftDeleteIncome(t *testing.T) {
	userID := uuid.New()
	incomeID := uuid.New()
	repo := &fakeIncomeStore{}
	svc := NewService(repo, nil, nil, nil)

	err := svc.SoftDeleteIncome(context.Background(), userID, incomeID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.deletedID != incomeID || repo.deletedUID != userID {
		t.Errorf("expected repo.SoftDeleteIncome called with %v and %v", userID, incomeID)
	}
}

func TestPagination(t *testing.T) {
	// 1. Empty pagination encoding
	emptyP := Pagination{}
	if encoded := emptyP.Encode(); encoded != "" {
		t.Errorf("expected empty string for zero/nil pagination fields, got %s", encoded)
	}

	// 2. Decode empty pagination
	dec, err := DecodePagination("")
	if err != nil {
		t.Fatalf("unexpected error decoding empty string: %v", err)
	}
	if dec != nil {
		t.Errorf("expected nil result decoding empty string")
	}

	// 3. Decode invalid pagination
	_, err = DecodePagination("invalid-base64-string!")
	if err == nil {
		t.Fatalf("expected error decoding invalid base64")
	}

	// 4. Encode & Decode success
	id := uuid.New()
	now := time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC)
	p := Pagination{
		CreatedAt: now,
		ID:        id,
	}

	encoded := p.Encode()
	if encoded == "" {
		t.Fatalf("expected non-empty base64 string")
	}

	decoded, err := DecodePagination(encoded)
	if err != nil {
		t.Fatalf("unexpected error decoding: %v", err)
	}
	if decoded.ID != id || !decoded.CreatedAt.Equal(now) {
		t.Errorf("expected decoded values to match, got ID %v and time %v", decoded.ID, decoded.CreatedAt)
	}
}

func TestServiceErrorPaths(t *testing.T) {
	fixedNow := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)

	t.Run("ListIncomesNilUser", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		_, _, err := svc.ListIncomes(context.Background(), uuid.Nil, 10, "")
		if err == nil {
			t.Errorf("expected error for nil userID")
		}
	})

	t.Run("ListIncomesInvalidPagination", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		_, _, err := svc.ListIncomes(context.Background(), uuid.New(), 10, "invalid-pagi!")
		if err == nil {
			t.Errorf("expected error for invalid pagination")
		}
	})

	t.Run("ListIncomesLimitClamping", func(t *testing.T) {
		repo := &fakeIncomeStore{}
		svc := NewService(repo, nil, nil, nil)

		// Negative limit → clamped to 20
		_, _, err := svc.ListIncomes(context.Background(), uuid.New(), -5, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Over 100 → clamped to 100
		_, _, err = svc.ListIncomes(context.Background(), uuid.New(), 200, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ListIncomesWithNextPagination", func(t *testing.T) {
		now := fixedNow
		nextID := uuid.New()
		repo := &fakeIncomeStore{
			listVal:  []*Income{{ID: uuid.New()}},
			listPage: &Pagination{CreatedAt: now, ID: nextID},
		}
		svc := NewService(repo, nil, nil, nil)

		_, next, err := svc.ListIncomes(context.Background(), uuid.New(), 1, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if next == "" {
			t.Errorf("expected non-empty next pagination token")
		}
	})

	t.Run("UpdateIncomeNilIDs", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		_, err := svc.UpdateIncome(context.Background(), uuid.Nil, uuid.New(), UpdateRequest{})
		if err == nil {
			t.Errorf("expected error for nil userID")
		}
		_, err = svc.UpdateIncome(context.Background(), uuid.New(), uuid.Nil, UpdateRequest{})
		if err == nil {
			t.Errorf("expected error for nil incomeID")
		}
	})

	t.Run("UpdateIncomeValidationError", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		svc.now = func() time.Time { return fixedNow }

		// Missing wallet_id
		_, err := svc.UpdateIncome(context.Background(), uuid.New(), uuid.New(), UpdateRequest{
			SourceName: "Test",
			Amount:     100,
			Currency:   "USD",
			IncomeDate: fixedNow,
		})
		if err == nil {
			t.Errorf("expected validation error for missing wallet")
		}
	})

	t.Run("UpdateIncomeGetNotFound", func(t *testing.T) {
		repo := &fakeIncomeStore{} // getVal is nil → returns not found
		svc := NewService(repo, nil, nil, nil)
		svc.now = func() time.Time { return fixedNow }

		_, err := svc.UpdateIncome(context.Background(), uuid.New(), uuid.New(), UpdateRequest{
			WalletID:   uuid.New(),
			SourceName: "Test",
			Amount:     100,
			Currency:   "USD",
			IncomeDate: fixedNow,
		})
		if err == nil {
			t.Errorf("expected not found error")
		}
	})

	t.Run("SoftDeleteIncomeNilIDs", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		err := svc.SoftDeleteIncome(context.Background(), uuid.Nil, uuid.New())
		if err == nil {
			t.Errorf("expected error for nil userID")
		}
		err = svc.SoftDeleteIncome(context.Background(), uuid.New(), uuid.Nil)
		if err == nil {
			t.Errorf("expected error for nil incomeID")
		}
	})

	t.Run("CreateIncomeCategoryForbidden", func(t *testing.T) {
		cg := fakeCategoryGuard{err: domain.NewDomainError(domain.ErrForbidden, "forbidden", 403)}
		svc := NewService(&fakeIncomeStore{}, nil, cg, nil)
		svc.now = func() time.Time { return fixedNow }

		catID := uuid.New()
		_, err := svc.CreateIncome(context.Background(), uuid.New(), CreateRequest{
			WalletID:   uuid.New(),
			CategoryID: &catID,
			SourceName: "Test",
			Amount:     100,
			Currency:   "USD",
			IncomeDate: fixedNow,
		})
		if err == nil {
			t.Errorf("expected category forbidden error")
		}
	})

	t.Run("NormalizeCurrencyNoServices", func(t *testing.T) {
		// When walletLookup and currencySvc are nil, currency defaults to uppercase or USD
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		svc.now = func() time.Time { return fixedNow }

		// Empty currency → defaults to USD
		inc, err := svc.CreateIncome(context.Background(), uuid.New(), CreateRequest{
			WalletID:   uuid.New(),
			SourceName: "Test",
			Amount:     100,
			IncomeDate: fixedNow,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inc.Currency != "USD" {
			t.Errorf("expected USD, got %s", inc.Currency)
		}
	})

	t.Run("NormalizeDateZero", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		svc.now = func() time.Time { return fixedNow }

		_, err := svc.CreateIncome(context.Background(), uuid.New(), CreateRequest{
			WalletID:   uuid.New(),
			SourceName: "Test",
			Amount:     100,
			Currency:   "USD",
			// IncomeDate is zero
		})
		if err == nil {
			t.Errorf("expected error for zero date")
		}
	})

	t.Run("NormalizeNotesEmpty", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		svc.now = func() time.Time { return fixedNow }

		emptyNotes := "   "
		inc, err := svc.CreateIncome(context.Background(), uuid.New(), CreateRequest{
			WalletID:   uuid.New(),
			SourceName: "Test",
			Amount:     100,
			Currency:   "USD",
			IncomeDate: fixedNow,
			Notes:      &emptyNotes,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inc.Notes != nil {
			t.Errorf("expected nil notes after trimming whitespace")
		}
	})

	t.Run("UpdateIncomeNilIDs", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		_, err := svc.UpdateIncome(context.Background(), uuid.Nil, uuid.Nil, UpdateRequest{})
		if err == nil {
			t.Errorf("expected error for nil IDs")
		}
	})

	t.Run("SoftDeleteIncomeNilIDs", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		err := svc.SoftDeleteIncome(context.Background(), uuid.Nil, uuid.Nil)
		if err == nil {
			t.Errorf("expected error for nil IDs")
		}
	})

	t.Run("ListIncomesLimitClamping", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		// limit=0 -> 20, limit=200 -> 100
		for _, limit := range []int{0, -1, 200} {
			_, _, err := svc.ListIncomes(context.Background(), uuid.New(), limit, "")
			if err != nil {
				t.Fatalf("unexpected error for limit %d: %v", limit, err)
			}
		}
	})

	t.Run("GetIncomeNilIDs", func(t *testing.T) {
		svc := NewService(&fakeIncomeStore{}, nil, nil, nil)
		_, err := svc.GetIncome(context.Background(), uuid.Nil, uuid.Nil)
		if err == nil {
			t.Errorf("expected error for nil IDs")
		}
	})

	t.Run("WalletNotFoundNormalization", func(t *testing.T) {
		wl := &fakeWalletLookup{wallet: nil} // nil wallet returned
		curSvc, _ := currency.NewService(nil)
		svc := NewService(&fakeIncomeStore{}, wl, nil, curSvc)
		svc.now = func() time.Time { return fixedNow }
		_, err := svc.CreateIncome(context.Background(), uuid.New(), CreateRequest{
			WalletID:   uuid.New(),
			SourceName: "Test",
			Amount:     100,
			IncomeDate: fixedNow,
			Currency:   "USD",
		})
		if err == nil {
			t.Errorf("expected error for wallet not found")
		}
	})
}

