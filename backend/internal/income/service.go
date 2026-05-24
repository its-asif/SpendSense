package income

import (
	"context"
	"strings"
	"time"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type Service struct {
	repo Store
	now  func() time.Time
}

func NewService(repo Store) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) CreateIncome(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Income, error) {
	if userID == uuid.Nil {
		return nil, domain.NewDomainError(domain.ErrUnauthorized, "user is required", 401)
	}

	validated, err := normalizeCreateRequest(req, s.now())
	if err != nil {
		return nil, err
	}

	income := &Income{
		ID:         uuid.New(),
		UserID:     userID,
		WalletID:   validated.WalletID,
		CategoryID: validated.CategoryID,
		SourceName: validated.SourceName,
		Amount:     validated.Amount,
		Currency:   validated.Currency,
		IncomeDate: validated.IncomeDate,
		Notes:      validated.Notes,
	}

	if err := s.repo.CreateIncome(ctx, income); err != nil {
		return nil, err
	}

	return income, nil
}

func (s *Service) GetIncome(ctx context.Context, userID, incomeID uuid.UUID) (*Income, error) {
	if userID == uuid.Nil || incomeID == uuid.Nil {
		return nil, domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
	}

	return s.repo.GetIncomeByID(ctx, userID, incomeID)
}

func (s *Service) ListIncomes(ctx context.Context, userID uuid.UUID, limit int, cursor string) ([]*Income, string, error) {
	if userID == uuid.Nil {
		return nil, "", domain.NewDomainError(domain.ErrUnauthorized, "user is required", 401)
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	decodedCursor, err := DecodeCursor(cursor)
	if err != nil {
		return nil, "", domain.NewDomainError(domain.ErrInvalidCursor, "invalid cursor", 400)
	}

	incomes, nextCursor, err := s.repo.ListIncomes(ctx, userID, limit, decodedCursor)
	if err != nil {
		return nil, "", err
	}

	if nextCursor == nil {
		return incomes, "", nil
	}

	return incomes, nextCursor.Encode(), nil
}

func (s *Service) UpdateIncome(ctx context.Context, userID, incomeID uuid.UUID, req UpdateRequest) (*Income, error) {
	if userID == uuid.Nil || incomeID == uuid.Nil {
		return nil, domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
	}

	validated, err := normalizeUpdateRequest(req, s.now())
	if err != nil {
		return nil, err
	}

	income, err := s.repo.GetIncomeByID(ctx, userID, incomeID)
	if err != nil {
		return nil, err
	}

	income.WalletID = validated.WalletID
	income.CategoryID = validated.CategoryID
	income.SourceName = validated.SourceName
	income.Amount = validated.Amount
	income.Currency = validated.Currency
	income.IncomeDate = validated.IncomeDate
	income.Notes = validated.Notes

	if err := s.repo.UpdateIncome(ctx, income); err != nil {
		return nil, err
	}

	return income, nil
}

func (s *Service) SoftDeleteIncome(ctx context.Context, userID, incomeID uuid.UUID) error {
	if userID == uuid.Nil || incomeID == uuid.Nil {
		return domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
	}

	return s.repo.SoftDeleteIncome(ctx, userID, incomeID)
}

type normalizedIncomeFields struct {
	WalletID   uuid.UUID
	CategoryID *uuid.UUID
	SourceName string
	Amount     float64
	Currency   string
	IncomeDate time.Time
	Notes      *string
}

func normalizeCreateRequest(req CreateRequest, now time.Time) (*normalizedIncomeFields, error) {
	return normalizeIncomeFields(normalizedIncomeFields{
		WalletID:   req.WalletID,
		CategoryID: req.CategoryID,
		SourceName: req.SourceName,
		Amount:     req.Amount,
		Currency:   req.Currency,
		IncomeDate: req.IncomeDate,
		Notes:      req.Notes,
	}, now)
}

func normalizeUpdateRequest(req UpdateRequest, now time.Time) (*normalizedIncomeFields, error) {
	return normalizeIncomeFields(normalizedIncomeFields{
		WalletID:   req.WalletID,
		CategoryID: req.CategoryID,
		SourceName: req.SourceName,
		Amount:     req.Amount,
		Currency:   req.Currency,
		IncomeDate: req.IncomeDate,
		Notes:      req.Notes,
	}, now)
}

func normalizeIncomeFields(normalized normalizedIncomeFields, now time.Time) (*normalizedIncomeFields, error) {
	if normalized.WalletID == uuid.Nil {
		return nil, domain.NewDomainError(domain.ErrInvalidWallet, "wallet_id is required", 400)
	}

	sourceName := strings.TrimSpace(normalized.SourceName)
	if sourceName == "" {
		return nil, domain.NewDomainError(domain.ErrNotFound, "source_name is required", 400)
	}
	normalized.SourceName = sourceName

	if normalized.Amount <= 0 {
		return nil, domain.NewDomainError(domain.ErrInvalidAmount, "amount must be greater than zero", 400)
	}

	dateOnly := stripTime(now)
	incomeDate := stripTime(normalized.IncomeDate)
	if incomeDate.IsZero() {
		return nil, domain.NewDomainError(domain.ErrInvalidDate, "income_date is required", 400)
	}
	if incomeDate.After(dateOnly) {
		return nil, domain.NewDomainError(domain.ErrInvalidDate, "income_date cannot be in the future", 400)
	}

	currency := strings.ToUpper(strings.TrimSpace(normalized.Currency))
	if currency == "" {
		currency = "USD"
	}
	if len(currency) != 3 {
		return nil, domain.NewDomainError(domain.ErrInvalidCurrency, "currency must be a 3-letter ISO code", 400)
	}
	normalized.Currency = currency

	if normalized.Notes != nil {
		trimmed := strings.TrimSpace(*normalized.Notes)
		if trimmed == "" {
			normalized.Notes = nil
		} else {
			normalized.Notes = &trimmed
		}
	}

	normalized.IncomeDate = incomeDate
	return &normalized, nil
}

func stripTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
