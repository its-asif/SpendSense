package expense

import (
	"context"
	"testing"
	"time"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type fakeExpenseStore struct {
	created        *Expense
	listFrom       *time.Time
	listTo         *time.Time
	listCategoryID *uuid.UUID
}

func (f *fakeExpenseStore) CreateExpense(ctx context.Context, e *Expense) error {
	f.created = e
	return nil
}
func (f *fakeExpenseStore) GetExpenseByID(ctx context.Context, userID, expenseID uuid.UUID) (*Expense, error) {
	return nil, nil
}
func (f *fakeExpenseStore) ListExpenses(ctx context.Context, userID uuid.UUID, limit int, pagination *Pagination, from, to *time.Time, categoryID *uuid.UUID) ([]*Expense, *Pagination, error) {
	f.listFrom = from
	f.listTo = to
	f.listCategoryID = categoryID
	return nil, nil, nil
}
func (f *fakeExpenseStore) UpdateExpense(ctx context.Context, expense *Expense) error { return nil }
func (f *fakeExpenseStore) SoftDeleteExpense(ctx context.Context, userID, expenseID uuid.UUID) error {
	return nil
}

func TestCreateExpenseValidation(t *testing.T) {
	repo := &fakeExpenseStore{}
	svc := NewService(repo, nil, nil)
	uid := uuid.New()

	// missing wallet
	_, err := svc.CreateExpense(context.Background(), uid, CreateRequest{WalletID: uuid.Nil, CategoryID: uuid.New(), Amount: 10, Date: time.Now()})
	if err == nil {
		t.Fatalf("expected error for missing wallet")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidWallet {
		t.Fatalf("expected ErrInvalidWallet, got %v", err)
	}

	// missing category
	_, err = svc.CreateExpense(context.Background(), uid, CreateRequest{WalletID: uuid.New(), CategoryID: uuid.Nil, Amount: 10, Date: time.Now()})
	if err == nil {
		t.Fatalf("expected error for missing category")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidCategory {
		t.Fatalf("expected ErrInvalidCategory, got %v", err)
	}

	// invalid amount
	_, err = svc.CreateExpense(context.Background(), uid, CreateRequest{WalletID: uuid.New(), CategoryID: uuid.New(), Amount: 0, Date: time.Now()})
	if err == nil {
		t.Fatalf("expected error for invalid amount")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}

	// future date
	future := time.Now().AddDate(0, 0, 1)
	_, err = svc.CreateExpense(context.Background(), uid, CreateRequest{WalletID: uuid.New(), CategoryID: uuid.New(), Amount: 10, Date: future})
	if err == nil {
		t.Fatalf("expected error for future date")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrInvalidDate {
		t.Fatalf("expected ErrInvalidDate, got %v", err)
	}

	// valid create
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
	svc := NewService(repo, nil, nil)
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
