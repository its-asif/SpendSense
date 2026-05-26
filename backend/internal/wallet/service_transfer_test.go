package wallet

import (
	"context"
	"testing"
	"time"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type fakeRepo struct {
	createCalled bool
	lastTransfer *Transfer
}

func (f *fakeRepo) CreateWallet(ctx context.Context, w *Wallet) error { return nil }
func (f *fakeRepo) GetWalletByID(ctx context.Context, userID, id uuid.UUID) (*Wallet, error) {
	return &Wallet{ID: id, UserID: userID, CurrentBalance: 1000, Currency: "USD"}, nil
}
func (f *fakeRepo) ListWallets(ctx context.Context, userID uuid.UUID) ([]*Wallet, error) {
	return nil, nil
}
func (f *fakeRepo) UpdateWallet(ctx context.Context, w *Wallet) error            { return nil }
func (f *fakeRepo) DeleteWallet(ctx context.Context, userID, id uuid.UUID) error { return nil }
func (f *fakeRepo) CreateTransfer(ctx context.Context, t *Transfer) error {
	f.createCalled = true
	f.lastTransfer = t
	return nil
}

func TestTransferValidation(t *testing.T) {
	repo := &fakeRepo{}
	s := NewService(repo, nil)
	userID := uuid.New()

	// invalid amount
	_, err := s.Transfer(context.Background(), userID, CreateTransferRequest{FromWalletID: uuid.New(), ToWalletID: uuid.New(), Amount: 0, FeeAmount: 0, Currency: "USD", TransferDate: time.Now().Format("2006-01-02")})
	if err == nil {
		t.Fatalf("expected error for zero amount")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}

	// same wallet
	id := uuid.New()
	_, err = s.Transfer(context.Background(), userID, CreateTransferRequest{FromWalletID: id, ToWalletID: id, Amount: 10, FeeAmount: 0, Currency: "USD", TransferDate: time.Now().Format("2006-01-02")})
	if err == nil {
		t.Fatalf("expected error for same wallet")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidWallet {
		t.Fatalf("expected ErrInvalidWallet, got %v", err)
	}

	// invalid date
	_, err = s.Transfer(context.Background(), userID, CreateTransferRequest{FromWalletID: uuid.New(), ToWalletID: uuid.New(), Amount: 10, FeeAmount: 0, Currency: "USD", TransferDate: "not-a-date"})
	if err == nil {
		t.Fatalf("expected error for invalid date")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidDate {
		t.Fatalf("expected ErrInvalidDate, got %v", err)
	}
}

func TestTransferCallsRepo(t *testing.T) {
	repo := &fakeRepo{}
	s := NewService(repo, nil)
	userID := uuid.New()
	from := uuid.New()
	to := uuid.New()
	td := time.Now().Format("2006-01-02")

	tResp, err := s.Transfer(context.Background(), userID, CreateTransferRequest{FromWalletID: from, ToWalletID: to, Amount: 10, FeeAmount: 1, Currency: "USD", TransferDate: td, Notes: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.createCalled {
		t.Fatalf("expected CreateTransfer called on repo")
	}
	if repo.lastTransfer == nil {
		t.Fatalf("lastTransfer nil")
	}
	if repo.lastTransfer.Amount != 10 || repo.lastTransfer.FeeAmount != 1 {
		t.Fatalf("unexpected transfer amounts: %+v", repo.lastTransfer)
	}
	if repo.lastTransfer.ExchangeRate != 1 || repo.lastTransfer.ConvertedAmount != 10 {
		t.Fatalf("unexpected conversion fields: %+v", repo.lastTransfer)
	}
	if tResp == nil {
		t.Fatalf("expected transfer response")
	}
}

func TestTransferAcceptsRFC3339TransferDate(t *testing.T) {
	repo := &fakeRepo{}
	s := NewService(repo, nil)
	userID := uuid.New()
	from := uuid.New()
	to := uuid.New()
	transferDate := "2026-05-25T20:52:34.613Z"

	_, err := s.Transfer(context.Background(), userID, CreateTransferRequest{FromWalletID: from, ToWalletID: to, Amount: 10, FeeAmount: 0, Currency: "USD", TransferDate: transferDate})
	if err != nil {
		t.Fatalf("unexpected error for RFC3339 transfer date: %v", err)
	}
	if repo.lastTransfer == nil {
		t.Fatalf("expected transfer to be recorded")
	}
	if got := repo.lastTransfer.TransferDate.Format("2006-01-02"); got != "2026-05-25" {
		t.Fatalf("expected normalized date 2026-05-25, got %s", got)
	}
}
