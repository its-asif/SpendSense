package category

import (
	"context"
	"testing"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type fakeCategoryStore struct {
	created    *Category
	updated    *Category
	deletedID  uuid.UUID
	deletedUID uuid.UUID
	getVal     *Category
	listVal    []*Category
	getErr     error
}

func (f *fakeCategoryStore) CreateCategory(ctx context.Context, userID uuid.UUID, c *Category) error {
	f.created = c
	return nil
}

func (f *fakeCategoryStore) GetCategoryByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID, kind string) (*Category, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getVal != nil {
		return f.getVal, nil
	}
	return nil, domain.NewDomainError(domain.ErrNotFound, "category not found", 404)
}

func (f *fakeCategoryStore) ListCategories(ctx context.Context, userID uuid.UUID, kind string) ([]*Category, error) {
	return f.listVal, nil
}

func (f *fakeCategoryStore) UpdateCategory(ctx context.Context, userID uuid.UUID, c *Category) error {
	f.updated = c
	return nil
}

func (f *fakeCategoryStore) DeleteCategory(ctx context.Context, userID, id uuid.UUID) error {
	f.deletedUID = userID
	f.deletedID = id
	return nil
}

func TestCategoryCreateValidation(t *testing.T) {
	repo := &fakeCategoryStore{}
	svc := NewService(repo)
	userID := uuid.New()

	// 1. Empty name
	_, err := svc.CreateCategory(context.Background(), userID, CreateRequest{Name: "  ", Kind: "EXPENSE"})
	if err == nil {
		t.Fatalf("expected error for empty category name")
	}

	// 2. Invalid kind
	_, err = svc.CreateCategory(context.Background(), userID, CreateRequest{Name: "Food", Kind: "INVALID"})
	if err == nil {
		t.Fatalf("expected error for invalid category kind")
	}

	// 3. Successful creation
	c, err := svc.CreateCategory(context.Background(), userID, CreateRequest{Name: "Salary ", Kind: "income"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Salary" || c.Kind != KindIncome {
		t.Errorf("unexpected category details: %+v", c)
	}
}

func TestCategoryGetAndList(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()
	expected := &Category{ID: catID, Name: "Rent", Kind: KindExpense}

	repo := &fakeCategoryStore{
		getVal:  expected,
		listVal: []*Category{expected},
	}
	svc := NewService(repo)

	// Get
	c, err := svc.GetCategory(context.Background(), userID, catID, "EXPENSE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Rent" {
		t.Errorf("expected Rent, got %s", c.Name)
	}

	// List
	list, err := svc.ListCategories(context.Background(), userID, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Rent" {
		t.Errorf("unexpected list output")
	}
}

func TestCategoryUpdateAndDelete(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()
	existingCustom := &Category{ID: catID, UserID: &userID, Name: "Entertainment", IsDefault: false}
	existingDefault := &Category{ID: catID, Name: "Food", IsDefault: true}

	repo := &fakeCategoryStore{getVal: existingCustom}
	svc := NewService(repo)

	// Update custom category
	updated, err := svc.UpdateCategory(context.Background(), userID, catID, UpdateRequest{Name: "Movies"})
	if err != nil {
		t.Fatalf("unexpected error updating category: %v", err)
	}
	if updated.Name != "Movies" {
		t.Errorf("expected name Movies, got %s", updated.Name)
	}

	// Try updating default category
	repo.getVal = existingDefault
	_, err = svc.UpdateCategory(context.Background(), userID, catID, UpdateRequest{Name: "Dinners"})
	if err == nil {
		t.Fatalf("expected error updating default category")
	}

	// Try deleting default category
	err = svc.DeleteCategory(context.Background(), userID, catID)
	if err == nil {
		t.Fatalf("expected error deleting default category")
	}

	// Delete custom category
	repo.getVal = existingCustom
	err = svc.DeleteCategory(context.Background(), userID, catID)
	if err != nil {
		t.Fatalf("unexpected error deleting category: %v", err)
	}
	if repo.deletedID != catID || repo.deletedUID != userID {
		t.Errorf("expected delete to be called on repository")
	}
}

func TestCategoryAssertAccessible(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()
	repo := &fakeCategoryStore{getVal: &Category{ID: catID}}
	svc := NewService(repo)

	err := svc.AssertAccessible(context.Background(), userID, catID, "EXPENSE")
	if err != nil {
		t.Fatalf("expected category to be accessible")
	}

	repo.getErr = domain.NewDomainError(domain.ErrNotFound, "not found", 404)
	err = svc.AssertAccessible(context.Background(), userID, catID, "EXPENSE")
	if err == nil {
		t.Fatalf("expected error for inaccessible category")
	}
}

func TestCategoryAdditionalPaths(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()

	t.Run("ListCategoriesInvalidKind", func(t *testing.T) {
		svc := NewService(&fakeCategoryStore{})
		_, err := svc.ListCategories(context.Background(), userID, "INVALID")
		if err == nil {
			t.Errorf("expected error for invalid kind")
		}
	})

	t.Run("ListCategoriesIncomeKind", func(t *testing.T) {
		repo := &fakeCategoryStore{listVal: []*Category{{ID: catID, Kind: KindIncome}}}
		svc := NewService(repo)
		list, err := svc.ListCategories(context.Background(), userID, "INCOME")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 1 {
			t.Errorf("expected 1 category, got %d", len(list))
		}
	})

	t.Run("AssertAccessibleInvalidKind", func(t *testing.T) {
		svc := NewService(&fakeCategoryStore{})
		err := svc.AssertAccessible(context.Background(), userID, catID, "INVALID")
		if err == nil {
			t.Errorf("expected error for invalid kind in AssertAccessible")
		}
	})

	t.Run("UpdateCategoryWrongOwner", func(t *testing.T) {
		otherUID := uuid.New()
		// Category belongs to a different user
		existingCustom := &Category{ID: catID, UserID: &otherUID, Name: "Groceries", IsDefault: false}
		repo := &fakeCategoryStore{getVal: existingCustom}
		svc := NewService(repo)
		_, err := svc.UpdateCategory(context.Background(), userID, catID, UpdateRequest{Name: "Updated"})
		if err == nil {
			t.Errorf("expected error for wrong owner on update")
		}
	})

	t.Run("CreateCategoryWithIconAndColor", func(t *testing.T) {
		repo := &fakeCategoryStore{}
		svc := NewService(repo)
		icon := "🍕"
		color := "#FF0000"
		c, err := svc.CreateCategory(context.Background(), userID, CreateRequest{
			Name:  "Pizza",
			Kind:  "EXPENSE",
			Icon:  &icon,
			Color: &color,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Icon == nil || *c.Icon != "🍕" {
			t.Errorf("expected icon to be set")
		}
		if c.Color == nil || *c.Color != "#FF0000" {
			t.Errorf("expected color to be set")
		}
	})

	t.Run("NormalizeKindEmpty", func(t *testing.T) {
		k, err := normalizeKind("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if k != KindExpense {
			t.Errorf("expected KindExpense default, got %s", k)
		}
	})
}
