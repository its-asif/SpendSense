package category

import (
	"context"
	"strings"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type Store interface {
	CreateCategory(ctx context.Context, userID uuid.UUID, c *Category) error
	GetCategoryByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID, kind string) (*Category, error)
	ListCategories(ctx context.Context, userID uuid.UUID, kind string) ([]*Category, error)
	UpdateCategory(ctx context.Context, userID uuid.UUID, c *Category) error
	DeleteCategory(ctx context.Context, userID, id uuid.UUID) error
}

type Service struct{ repo Store }

func NewService(r Store) *Service { return &Service{repo: r} }

func (s *Service) CreateCategory(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Category, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidCategory, "name required", 400)
	}
	kind, err := normalizeKind(req.Kind)
	if err != nil {
		return nil, err
	}

	c := &Category{
		ID:    uuid.New(),
		Name:  strings.TrimSpace(req.Name),
		Icon:  req.Icon,
		Color: req.Color,
		Kind:  kind,
	}
	if err := s.repo.CreateCategory(ctx, userID, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) GetCategory(ctx context.Context, userID, id uuid.UUID, kind string) (*Category, error) {
	return s.repo.GetCategoryByID(ctx, id, &userID, kind)
}

func (s *Service) ListCategories(ctx context.Context, userID uuid.UUID, kind string) ([]*Category, error) {
	normalized, err := normalizeKind(kind)
	if err != nil && kind != "" {
		return nil, err
	}
	if kind == "" {
		normalized = KindExpense
	}
	return s.repo.ListCategories(ctx, userID, normalized)
}

func (s *Service) UpdateCategory(ctx context.Context, userID uuid.UUID, id uuid.UUID, req UpdateRequest) (*Category, error) {
	existing, err := s.repo.GetCategoryByID(ctx, id, &userID, "")
	if err != nil {
		return nil, err
	}
	if existing.IsDefault {
		return nil, domain.NewDomainError(domain.ErrForbidden, "default categories cannot be edited", 403)
	}
	if existing.UserID == nil || *existing.UserID != userID {
		return nil, domain.NewDomainError(domain.ErrNotFound, "category not found", 404)
	}

	existing.Name = strings.TrimSpace(req.Name)
	existing.Icon = req.Icon
	existing.Color = req.Color
	if err := s.repo.UpdateCategory(ctx, userID, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *Service) AssertAccessible(ctx context.Context, userID, categoryID uuid.UUID, kind string) error {
	normalized, err := normalizeKind(kind)
	if err != nil {
		return err
	}
	_, err = s.repo.GetCategoryByID(ctx, categoryID, &userID, normalized)
	return err
}

func (s *Service) DeleteCategory(ctx context.Context, userID, id uuid.UUID) error {
	existing, err := s.repo.GetCategoryByID(ctx, id, &userID, "")
	if err != nil {
		return err
	}
	if existing.IsDefault {
		return domain.NewDomainError(domain.ErrForbidden, "default categories cannot be deleted", 403)
	}
	return s.repo.DeleteCategory(ctx, userID, id)
}

func normalizeKind(kind string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(kind))
	if normalized == "" {
		return KindExpense, nil
	}
	switch normalized {
	case KindExpense, KindIncome:
		return normalized, nil
	default:
		return "", domain.NewDomainErrorWithField(domain.ErrInvalidCategory, "kind must be EXPENSE or INCOME", "kind", 400)
	}
}
