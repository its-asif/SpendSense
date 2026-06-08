package budget

import (
	"context"
	"testing"
	"time"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type fakeBudgetStore struct {
	created          *Budget
	updated          *Budget
	deletedID        uuid.UUID
	deletedUID       uuid.UUID
	getVal           *Budget
	listVal          []*Budget
	isAccessible     bool
	monthlyExists    bool
	accessibleErr    error
	monthlyExistsErr error
}

func (f *fakeBudgetStore) Create(ctx context.Context, b *Budget) error {
	f.created = b
	b.CreatedAt = time.Now()
	b.UpdatedAt = time.Now()
	return nil
}

func (f *fakeBudgetStore) GetByID(ctx context.Context, userID, id uuid.UUID) (*Budget, error) {
	if f.getVal != nil {
		return f.getVal, nil
	}
	if f.created != nil && f.created.ID == id {
		return f.created, nil
	}
	return nil, domain.NewDomainError(domain.ErrNotFound, "budget not found", 404)
}

func (f *fakeBudgetStore) List(ctx context.Context, userID uuid.UUID, period string) ([]*Budget, error) {
	return f.listVal, nil
}

func (f *fakeBudgetStore) Update(ctx context.Context, b *Budget) error {
	f.updated = b
	return nil
}

func (f *fakeBudgetStore) Delete(ctx context.Context, userID, id uuid.UUID) error {
	f.deletedUID = userID
	f.deletedID = id
	return nil
}

func (f *fakeBudgetStore) CategoryAccessible(ctx context.Context, userID, categoryID uuid.UUID) (bool, error) {
	return f.isAccessible, f.accessibleErr
}

func (f *fakeBudgetStore) HasMonthlyBudgetForCategory(ctx context.Context, userID, categoryID uuid.UUID, excludeID *uuid.UUID) (bool, error) {
	return f.monthlyExists, f.monthlyExistsErr
}

func TestBudgetCreateValidation(t *testing.T) {
	repo := &fakeBudgetStore{isAccessible: true}
	svc := NewService(repo, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	}

	userID := uuid.New()

	// 1. Missing user ID
	_, err := svc.Create(context.Background(), uuid.Nil, CreateRequest{})
	if err == nil {
		t.Fatalf("expected error for empty user ID")
	}

	// 2. Missing category ID
	_, err = svc.Create(context.Background(), userID, CreateRequest{
		Amount: 100,
	})
	if err == nil {
		t.Fatalf("expected error for empty category ID")
	}

	// 3. Invalid amount
	_, err = svc.Create(context.Background(), userID, CreateRequest{
		CategoryID: uuid.New(),
		Amount:     -10.5,
	})
	if err == nil {
		t.Fatalf("expected error for negative budget amount")
	}

	// 4. Invalid period
	_, err = svc.Create(context.Background(), userID, CreateRequest{
		CategoryID: uuid.New(),
		Amount:     100,
		Period:     "DAILY",
	})
	if err == nil {
		t.Fatalf("expected error for invalid period")
	}

	// 5. Category not accessible
	repo.isAccessible = false
	_, err = svc.Create(context.Background(), userID, CreateRequest{
		CategoryID: uuid.New(),
		Amount:     100,
		Period:     "MONTHLY",
	})
	if err == nil {
		t.Fatalf("expected error for inaccessible category")
	}
	repo.isAccessible = true

	// 6. Monthly budget duplicate
	repo.monthlyExists = true
	_, err = svc.Create(context.Background(), userID, CreateRequest{
		CategoryID: uuid.New(),
		Amount:     100,
		Period:     "MONTHLY",
	})
	if err == nil {
		t.Fatalf("expected error for duplicate monthly budget")
	}
	repo.monthlyExists = false

	// 7. Successful creation
	catID := uuid.New()
	budget, err := svc.Create(context.Background(), userID, CreateRequest{
		CategoryID:      catID,
		Amount:          250.778,
		Period:          "MONTHLY",
		RolloverEnabled: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if budget == nil {
		t.Fatalf("expected budget to be returned")
	}
	if budget.Amount != 250.78 {
		t.Errorf("expected rounded amount 250.78, got %f", budget.Amount)
	}
}

func TestBudgetGetAndList(t *testing.T) {
	userID := uuid.New()
	budgetID := uuid.New()
	expected := &Budget{ID: budgetID, UserID: userID, Amount: 100, Period: "MONTHLY"}

	repo := &fakeBudgetStore{
		getVal:  expected,
		listVal: []*Budget{expected},
	}
	svc := NewService(repo, nil)

	// Get
	b, err := svc.Get(context.Background(), userID, budgetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Amount != 100 {
		t.Errorf("expected amount 100, got %f", b.Amount)
	}

	// List
	list, err := svc.List(context.Background(), userID, "monthly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].Period != "MONTHLY" {
		t.Errorf("unexpected list output")
	}
}

func TestBudgetUpdateAndDelete(t *testing.T) {
	userID := uuid.New()
	budgetID := uuid.New()
	existing := &Budget{ID: budgetID, UserID: userID, Amount: 100, Period: "MONTHLY"}

	repo := &fakeBudgetStore{
		getVal:       existing,
		isAccessible: true,
	}
	svc := NewService(repo, nil)
	svc.now = func() time.Time {
		return time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	}

	// Update
	updated, err := svc.Update(context.Background(), userID, budgetID, UpdateRequest{
		CategoryID:      uuid.New(),
		Amount:          150,
		Period:          "MONTHLY",
		RolloverEnabled: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Amount != 150 {
		t.Errorf("expected updated amount 150, got %f", updated.Amount)
	}

	// Delete
	err = svc.Delete(context.Background(), userID, budgetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.deletedID != budgetID || repo.deletedUID != userID {
		t.Errorf("expected delete to be called on repository")
	}
}
