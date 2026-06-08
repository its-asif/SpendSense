package wallet

import (
	"context"
	"testing"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type fakeWalletCRUDRepo struct {
	created    *Wallet
	updated    *Wallet
	deletedID  uuid.UUID
	deletedUID uuid.UUID
	getVal     *Wallet
	listVal    []*Wallet
	getErr     error
	updateErr  error
	createErr  error
}

func (f *fakeWalletCRUDRepo) CreateWallet(ctx context.Context, w *Wallet) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = w
	return nil
}

func (f *fakeWalletCRUDRepo) GetWalletByID(ctx context.Context, userID, id uuid.UUID) (*Wallet, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getVal != nil {
		return f.getVal, nil
	}
	return &Wallet{ID: id, UserID: userID, Name: "Test Wallet", Currency: "USD"}, nil
}

func (f *fakeWalletCRUDRepo) ListWallets(ctx context.Context, userID uuid.UUID) ([]*Wallet, error) {
	return f.listVal, nil
}

func (f *fakeWalletCRUDRepo) UpdateWallet(ctx context.Context, w *Wallet) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = w
	return nil
}

func (f *fakeWalletCRUDRepo) DeleteWallet(ctx context.Context, userID, id uuid.UUID) error {
	f.deletedUID = userID
	f.deletedID = id
	return nil
}

func (f *fakeWalletCRUDRepo) CreateTransfer(ctx context.Context, t *Transfer) error {
	return nil
}

func TestCreateWalletValidation(t *testing.T) {
	repo := &fakeWalletCRUDRepo{}
	svc := NewService(repo, nil)
	userID := uuid.New()

	// 1. Missing name
	_, err := svc.CreateWallet(context.Background(), userID, CreateRequest{Name: "", Currency: "USD"})
	if err == nil {
		t.Fatalf("expected error for empty name")
	}

	// 2. Successful creation
	w, err := svc.CreateWallet(context.Background(), userID, CreateRequest{
		Name:           "Personal Cash",
		WalletType:     "CASH",
		Currency:       " eur ",
		OpeningBalance: 50.123,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Name != "Personal Cash" || w.Currency != "EUR" || w.OpeningBalance != 50.12 {
		t.Errorf("unexpected wallet details: %+v", w)
	}
	if repo.created != w {
		t.Errorf("expected wallet to be created in repository")
	}
}

func TestGetAndListWallets(t *testing.T) {
	userID := uuid.New()
	walletID := uuid.New()
	expected := &Wallet{ID: walletID, UserID: userID, Name: "Checking", Currency: "USD"}

	repo := &fakeWalletCRUDRepo{
		getVal:  expected,
		listVal: []*Wallet{expected},
	}
	svc := NewService(repo, nil)

	// Get
	w, err := svc.GetWallet(context.Background(), userID, walletID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Name != "Checking" {
		t.Errorf("expected Checking, got %s", w.Name)
	}

	// List
	list, err := svc.ListWallets(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Checking" {
		t.Errorf("unexpected list output")
	}
}

func TestUpdateAndDeleteWallet(t *testing.T) {
	userID := uuid.New()
	walletID := uuid.New()
	existing := &Wallet{ID: walletID, UserID: userID, Name: "Savings", Currency: "USD", CurrentBalance: 200}

	repo := &fakeWalletCRUDRepo{getVal: existing}
	svc := NewService(repo, nil)

	// Update
	updated, err := svc.UpdateWallet(context.Background(), userID, walletID, UpdateRequest{
		Name:           "High Yield Savings",
		WalletType:     "SAVINGS",
		Currency:       "USD",
		IsActive:       true,
		CurrentBalance: 250.556,
	})
	if err != nil {
		t.Fatalf("unexpected error updating wallet: %v", err)
	}
	if updated.Name != "High Yield Savings" || updated.CurrentBalance != 250.56 {
		t.Errorf("unexpected updated wallet details: %+v", updated)
	}

	// Delete
	err = svc.DeleteWallet(context.Background(), userID, walletID)
	if err != nil {
		t.Fatalf("unexpected error deleting wallet: %v", err)
	}
	if repo.deletedID != walletID || repo.deletedUID != userID {
		t.Errorf("expected delete to be called on repository")
	}
}

func TestWalletServiceAdditionalPaths(t *testing.T) {
	userID := uuid.New()

	t.Run("CreateWalletRepoError", func(t *testing.T) {
		repo := &fakeWalletCRUDRepo{createErr: domain.NewDomainError(domain.ErrInternal, "db error", 500)}
		svc := NewService(repo, nil)
		_, err := svc.CreateWallet(context.Background(), userID, CreateRequest{Name: "Test", Currency: "USD"})
		if err == nil {
			t.Errorf("expected repo error")
		}
	})

	t.Run("GetWalletError", func(t *testing.T) {
		repo := &fakeWalletCRUDRepo{getErr: domain.NewDomainError(domain.ErrNotFound, "not found", 404)}
		svc := NewService(repo, nil)
		_, err := svc.GetWallet(context.Background(), userID, uuid.New())
		if err == nil {
			t.Errorf("expected not found error")
		}
	})

	t.Run("UpdateWalletGetError", func(t *testing.T) {
		repo := &fakeWalletCRUDRepo{getErr: domain.NewDomainError(domain.ErrNotFound, "not found", 404)}
		svc := NewService(repo, nil)
		_, err := svc.UpdateWallet(context.Background(), userID, uuid.New(), UpdateRequest{Name: "x"})
		if err == nil {
			t.Errorf("expected not found error")
		}
	})

	t.Run("UpdateWalletNoCurrencyChange", func(t *testing.T) {
		walletID := uuid.New()
		existing := &Wallet{ID: walletID, UserID: userID, Name: "Savings", Currency: "USD"}
		repo := &fakeWalletCRUDRepo{getVal: existing}
		svc := NewService(repo, nil)
		// No currency in request → currency stays unchanged
		updated, err := svc.UpdateWallet(context.Background(), userID, walletID, UpdateRequest{Name: "New Name"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Currency != "USD" {
			t.Errorf("expected USD currency unchanged, got %s", updated.Currency)
		}
	})

	t.Run("RoundCurrencyCode", func(t *testing.T) {
		if got := roundCurrencyCode(""); got != "" {
			t.Errorf("expected empty, got %s", got)
		}
		if got := roundCurrencyCode(" eur "); got != "EUR" {
			t.Errorf("expected EUR, got %s", got)
		}
	})
}

func TestTransferAdditionalPaths(t *testing.T) {
	userID := uuid.New()

	t.Run("NegativeFeeAmount", func(t *testing.T) {
		repo := &fakeRepo{}
		s := NewService(repo, nil)
		_, err := s.Transfer(context.Background(), userID, CreateTransferRequest{
			FromWalletID: uuid.New(),
			ToWalletID:   uuid.New(),
			Amount:       10,
			FeeAmount:    -1,
			TransferDate: "2026-06-07",
		})
		if err == nil {
			t.Errorf("expected error for negative fee")
		}
	})

	t.Run("MismatchedCurrency", func(t *testing.T) {
		repo := &fakeRepo{}
		s := NewService(repo, nil)
		_, err := s.Transfer(context.Background(), userID, CreateTransferRequest{
			FromWalletID: uuid.New(),
			ToWalletID:   uuid.New(),
			Amount:       10,
			Currency:     "EUR", // wallet currency is USD
			TransferDate: "2026-06-07",
		})
		if err == nil {
			t.Errorf("expected currency mismatch error")
		}
	})

	t.Run("CrossCurrencyExplicitRate", func(t *testing.T) {
		// fakeRepo returns different currencies based on ID
		from := uuid.New()
		to := uuid.New()
		repo := &fakeRepoMultiCurrency{fromID: from, toID: to}
		s := NewService(repo, nil)

		tr, err := s.Transfer(context.Background(), userID, CreateTransferRequest{
			FromWalletID: from,
			ToWalletID:   to,
			Amount:       100,
			ExchangeRate: 0.85,
			TransferDate: "2026-06-07",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr.ExchangeRate != 0.85 || tr.ConvertedAmount != 85 {
			t.Errorf("unexpected transfer: %+v", tr)
		}
	})

	t.Run("CrossCurrencyNoRateNoCurrencySvc", func(t *testing.T) {
		from := uuid.New()
		to := uuid.New()
		repo := &fakeRepoMultiCurrency{fromID: from, toID: to}
		s := NewService(repo, nil) // no currency service
		_, err := s.Transfer(context.Background(), userID, CreateTransferRequest{
			FromWalletID: from,
			ToWalletID:   to,
			Amount:       100,
			ExchangeRate: 0, // missing rate, no svc
			TransferDate: "2026-06-07",
		})
		if err == nil {
			t.Errorf("expected error: exchange_rate required without currency service")
		}
	})

	t.Run("EmptyTransferDate", func(t *testing.T) {
		repo := &fakeRepo{}
		s := NewService(repo, nil)
		_, err := s.Transfer(context.Background(), userID, CreateTransferRequest{
			FromWalletID: uuid.New(),
			ToWalletID:   uuid.New(),
			Amount:       10,
			TransferDate: "",
		})
		if err == nil {
			t.Errorf("expected error for empty transfer date")
		}
	})
}

type fakeRepoMultiCurrency struct {
	fromID uuid.UUID
	toID   uuid.UUID
	createCalled bool
}

func (f *fakeRepoMultiCurrency) CreateWallet(ctx context.Context, w *Wallet) error { return nil }
func (f *fakeRepoMultiCurrency) GetWalletByID(ctx context.Context, userID, id uuid.UUID) (*Wallet, error) {
	if id == f.fromID {
		return &Wallet{ID: id, Currency: "USD"}, nil
	}
	return &Wallet{ID: id, Currency: "EUR"}, nil
}
func (f *fakeRepoMultiCurrency) ListWallets(ctx context.Context, userID uuid.UUID) ([]*Wallet, error) {
	return nil, nil
}
func (f *fakeRepoMultiCurrency) UpdateWallet(ctx context.Context, w *Wallet) error            { return nil }
func (f *fakeRepoMultiCurrency) DeleteWallet(ctx context.Context, userID, id uuid.UUID) error { return nil }
func (f *fakeRepoMultiCurrency) CreateTransfer(ctx context.Context, t *Transfer) error {
	f.createCalled = true
	return nil
}
