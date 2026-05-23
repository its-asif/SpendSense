package category

import (
	"context"
	"time"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type Service struct{ repo *Repository }

func NewService(r *Repository) *Service { return &Service{repo: r} }

func (s *Service) CreateCategory(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Category, error) {
	if req.Name == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidCategory, "name required", 400)
	}
	c := &Category{ID: uuid.New(), Name: req.Name, Icon: req.Icon, Color: req.Color, IsDefault: false, CreatedAt: time.Time{}}
	if err := s.repo.CreateCategory(ctx, userID, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) GetCategory(ctx context.Context, id uuid.UUID) (*Category, error) {
	c, err := s.repo.GetCategoryByID(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) ListCategories(ctx context.Context, userID uuid.UUID) ([]*Category, error) {
	return s.repo.ListCategories(ctx, userID)
}

func (s *Service) UpdateCategory(ctx context.Context, userID uuid.UUID, id uuid.UUID, req UpdateRequest) (*Category, error) {
	c, err := s.repo.GetCategoryByID(ctx, id, &userID)
	if err != nil {
		return nil, err
	}
	c.Name = req.Name
	c.Icon = req.Icon
	c.Color = req.Color
	if err := s.repo.UpdateCategory(ctx, userID, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) DeleteCategory(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.DeleteCategory(ctx, userID, id)
}
