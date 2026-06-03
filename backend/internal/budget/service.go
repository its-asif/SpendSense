package budget

import (
	"context"
	"math"
	"strings"
	"time"

	"spendsense-backend/internal/currency"
	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type Service struct {
	repo        *Repository
	currencySvc *currency.Service
	now         func() time.Time
}

func NewService(repo *Repository, currencySvc *currency.Service) *Service {
	return &Service{repo: repo, currencySvc: currencySvc, now: time.Now}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Budget, error) {
	normalized, err := s.normalizeRequest(userID, req)
	if err != nil {
		return nil, err
	}

	if err := s.ensureCategoryAccess(ctx, userID, normalized.CategoryID); err != nil {
		return nil, err
	}
	if err := s.ensureUniqueMonthly(ctx, userID, normalized.CategoryID, normalized.Period, nil); err != nil {
		return nil, err
	}

	b := &Budget{
		ID:              uuid.New(),
		UserID:          userID,
		CategoryID:      normalized.CategoryID,
		Amount:          normalized.Amount,
		Currency:        normalized.Currency,
		Period:          normalized.Period,
		StartDate:       normalized.StartDate,
		RolloverEnabled: normalized.RolloverEnabled,
	}
	if err := s.repo.Create(ctx, b); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, userID, b.ID)
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (*Budget, error) {
	return s.repo.GetByID(ctx, userID, id)
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, period string) ([]*Budget, error) {
	period = strings.ToUpper(strings.TrimSpace(period))
	return s.repo.List(ctx, userID, period)
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Budget, error) {
	existing, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	normalized, err := s.normalizeRequest(userID, CreateRequest{
		CategoryID:      req.CategoryID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Period:          req.Period,
		StartDate:       req.StartDate,
		RolloverEnabled: req.RolloverEnabled,
	})
	if err != nil {
		return nil, err
	}

	if err := s.ensureCategoryAccess(ctx, userID, normalized.CategoryID); err != nil {
		return nil, err
	}
	if err := s.ensureUniqueMonthly(ctx, userID, normalized.CategoryID, normalized.Period, &id); err != nil {
		return nil, err
	}

	existing.CategoryID = normalized.CategoryID
	existing.Amount = normalized.Amount
	existing.Currency = normalized.Currency
	existing.Period = normalized.Period
	existing.StartDate = normalized.StartDate
	existing.RolloverEnabled = normalized.RolloverEnabled

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, userID, id)
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}

func (s *Service) normalizeRequest(userID uuid.UUID, req CreateRequest) (CreateRequest, error) {
	if userID == uuid.Nil {
		return req, domain.NewDomainError(domain.ErrUnauthorized, "user is required", 401)
	}
	if req.CategoryID == uuid.Nil {
		return req, domain.NewDomainErrorWithField(domain.ErrInvalidCategory, "category is required", "category_id", 400)
	}
	if req.Amount <= 0 || math.IsNaN(req.Amount) || math.IsInf(req.Amount, 0) {
		return req, domain.NewDomainErrorWithField(domain.ErrInvalidAmount, "amount must be greater than zero", "amount", 400)
	}

	period := strings.ToUpper(strings.TrimSpace(req.Period))
	if period == "" {
		period = PeriodMonthly
	}
	switch period {
	case PeriodMonthly, PeriodWeekly, PeriodYearly:
	default:
		return req, domain.NewDomainErrorWithField(domain.ErrInvalidDate, "period must be MONTHLY, WEEKLY, or YEARLY", "period", 400)
	}

	startDate := req.StartDate
	if startDate.IsZero() {
		now := s.now().UTC()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	currencyCode := strings.ToUpper(strings.TrimSpace(req.Currency))
	if s.currencySvc != nil {
		resolved, err := s.currencySvc.NormalizeOrDefault(currencyCode, "", nil)
		if err != nil {
			return req, domain.NewDomainError(domain.ErrInvalidCurrency, err.Error(), 400)
		}
		currencyCode = resolved
	} else if currencyCode == "" {
		currencyCode = "USD"
	}

	req.Amount = round2(req.Amount)
	req.Currency = currencyCode
	req.Period = period
	req.StartDate = startDate
	return req, nil
}

func (s *Service) ensureCategoryAccess(ctx context.Context, userID, categoryID uuid.UUID) error {
	ok, err := s.repo.CategoryAccessible(ctx, userID, categoryID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.NewDomainError(domain.ErrInvalidCategory, "category not found", 404)
	}
	return nil
}

func (s *Service) ensureUniqueMonthly(ctx context.Context, userID, categoryID uuid.UUID, period string, excludeID *uuid.UUID) error {
	if strings.ToUpper(period) != PeriodMonthly {
		return nil
	}
	exists, err := s.repo.HasMonthlyBudgetForCategory(ctx, userID, categoryID, excludeID)
	if err != nil {
		return err
	}
	if exists {
		return domain.NewDomainError(domain.ErrAlreadyExists, "a monthly budget already exists for this category", 409)
	}
	return nil
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
