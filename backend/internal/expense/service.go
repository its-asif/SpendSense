package expense

import (
	"context"
	"math"
	"strings"
	"time"

	"spendsense-backend/internal/currency"
	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/wallet"

	"github.com/google/uuid"
)

type Service struct {
	repo         Store
	walletLookup walletLookup
	currencySvc  *currency.Service
	now          func() time.Time
}

type walletLookup interface {
	GetWalletByID(ctx context.Context, userID, id uuid.UUID) (*wallet.Wallet, error)
}

func NewService(repo Store, walletLookup walletLookup, currencySvc *currency.Service) *Service {
	return &Service{repo: repo, walletLookup: walletLookup, currencySvc: currencySvc, now: time.Now}
}

func (s *Service) CreateExpense(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Expense, error) {
	if userID == uuid.Nil {
		return nil, domain.NewDomainError(domain.ErrUnauthorized, "user is required", 401)
	}

	validated, err := normalizeCreateRequest(req, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.normalizeCurrency(ctx, userID, validated); err != nil {
		return nil, err
	}

	expense := &Expense{
		ID:            uuid.New(),
		UserID:        userID,
		WalletID:      validated.WalletID,
		Amount:        validated.Amount,
		Currency:      validated.Currency,
		FXRateToBase:  validated.FXRateToBase,
		CategoryID:    validated.CategoryID,
		Merchant:      validated.Merchant,
		Date:          validated.Date,
		Notes:         validated.Notes,
		IsRecurring:   validated.IsRecurring,
		RecurringRule: validated.RecurringRule,
	}

	if err := s.repo.CreateExpense(ctx, expense); err != nil {
		return nil, err
	}

	return expense, nil
}

func (s *Service) GetExpense(ctx context.Context, userID, expenseID uuid.UUID) (*Expense, error) {
	if userID == uuid.Nil || expenseID == uuid.Nil {
		return nil, domain.NewDomainError(domain.ErrNotFound, "expense not found", 404)
	}

	return s.repo.GetExpenseByID(ctx, userID, expenseID)
}

func (s *Service) ListExpenses(ctx context.Context, userID uuid.UUID, limit int, pagination string, from, to *time.Time, categoryID *uuid.UUID) ([]*Expense, string, error) {
	if userID == uuid.Nil {
		return nil, "", domain.NewDomainError(domain.ErrUnauthorized, "user is required", 401)
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	decodedPagination, err := DecodePagination(pagination)
	if err != nil {
		return nil, "", domain.NewDomainError(domain.ErrInvalidPagination, "invalid pagination", 400)
	}

	expenses, nextPagination, err := s.repo.ListExpenses(ctx, userID, limit, decodedPagination, from, to, categoryID)
	if err != nil {
		return nil, "", err
	}

	if nextPagination == nil {
		return expenses, "", nil
	}

	return expenses, nextPagination.Encode(), nil
}

func (s *Service) UpdateExpense(ctx context.Context, userID, expenseID uuid.UUID, req UpdateRequest) (*Expense, error) {
	if userID == uuid.Nil || expenseID == uuid.Nil {
		return nil, domain.NewDomainError(domain.ErrNotFound, "expense not found", 404)
	}

	validated, err := normalizeUpdateRequest(req, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.normalizeCurrency(ctx, userID, validated); err != nil {
		return nil, err
	}

	expense := &Expense{
		ID:            expenseID,
		UserID:        userID,
		WalletID:      validated.WalletID,
		Amount:        validated.Amount,
		Currency:      validated.Currency,
		FXRateToBase:  validated.FXRateToBase,
		CategoryID:    validated.CategoryID,
		Merchant:      validated.Merchant,
		Date:          validated.Date,
		Notes:         validated.Notes,
		IsRecurring:   validated.IsRecurring,
		RecurringRule: validated.RecurringRule,
	}

	if err := s.repo.UpdateExpense(ctx, expense); err != nil {
		return nil, err
	}

	return expense, nil
}

func (s *Service) SoftDeleteExpense(ctx context.Context, userID, expenseID uuid.UUID) error {
	if userID == uuid.Nil || expenseID == uuid.Nil {
		return domain.NewDomainError(domain.ErrNotFound, "expense not found", 404)
	}

	return s.repo.SoftDeleteExpense(ctx, userID, expenseID)
}

type normalizedExpenseFields struct {
	WalletID      uuid.UUID
	Amount        float64
	Currency      string
	FXRateToBase  float64
	CategoryID    uuid.UUID
	Merchant      *string
	Date          time.Time
	Notes         *string
	IsRecurring   bool
	RecurringRule *string
}

func (s *Service) normalizeCurrency(ctx context.Context, userID uuid.UUID, normalized *normalizedExpenseFields) error {
	if s.walletLookup == nil || s.currencySvc == nil {
		normalized.Currency = strings.ToUpper(strings.TrimSpace(normalized.Currency))
		if normalized.Currency == "" {
			normalized.Currency = "USD"
		}
		return nil
	}

	walletObj, err := s.walletLookup.GetWalletByID(ctx, userID, normalized.WalletID)
	if err != nil {
		return err
	}
	if walletObj == nil {
		return domain.NewDomainError(domain.ErrNotFound, "wallet not found", 404)
	}

	walletCurrency, err := s.currencySvc.NormalizeOrDefault(walletObj.Currency, walletObj.WalletType, walletObj.Provider)
	if err != nil {
		return domain.NewDomainError(domain.ErrInvalidCurrency, err.Error(), 400)
	}

	sourceCurrency := strings.ToUpper(strings.TrimSpace(normalized.Currency))
	if sourceCurrency == "" {
		sourceCurrency = walletCurrency
	}
	if sourceCurrency != walletCurrency {
		convertedAmount, rate, err := s.currencySvc.Convert(ctx, normalized.Amount, sourceCurrency, walletCurrency)
		if err != nil {
			return domain.NewDomainError(domain.ErrInvalidCurrency, err.Error(), 400)
		}
		normalized.Amount = convertedAmount
		normalized.FXRateToBase = rate
	} else {
		normalized.FXRateToBase = 1
	}

	normalized.Amount = round2(normalized.Amount)
	normalized.FXRateToBase = round2(normalized.FXRateToBase)
	normalized.Currency = walletCurrency
	return nil
}

func normalizeCreateRequest(req CreateRequest, now time.Time) (*normalizedExpenseFields, error) {
	return normalizeExpenseFields(normalizedExpenseFields{
		WalletID:      req.WalletID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		FXRateToBase:  req.FXRateToBase,
		CategoryID:    req.CategoryID,
		Merchant:      req.Merchant,
		Date:          req.Date,
		Notes:         req.Notes,
		IsRecurring:   req.IsRecurring,
		RecurringRule: req.RecurringRule,
	}, now)
}

func normalizeUpdateRequest(req UpdateRequest, now time.Time) (*normalizedExpenseFields, error) {
	return normalizeExpenseFields(normalizedExpenseFields{
		WalletID:      req.WalletID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		FXRateToBase:  req.FXRateToBase,
		CategoryID:    req.CategoryID,
		Merchant:      req.Merchant,
		Date:          req.Date,
		Notes:         req.Notes,
		IsRecurring:   req.IsRecurring,
		RecurringRule: req.RecurringRule,
	}, now)
}

func normalizeExpenseFields(normalized normalizedExpenseFields, now time.Time) (*normalizedExpenseFields, error) {

	if normalized.WalletID == uuid.Nil {
		return nil, domain.NewDomainError(domain.ErrInvalidWallet, "wallet_id is required", 400)
	}
	if normalized.CategoryID == uuid.Nil {
		return nil, domain.NewDomainError(domain.ErrInvalidCategory, "category_id is required", 400)
	}
	if normalized.Amount <= 0 {
		return nil, domain.NewDomainError(domain.ErrInvalidAmount, "amount must be greater than zero", 400)
	}
	normalized.Amount = round2(normalized.Amount)

	dateOnly := stripTime(now)
	expenseDate := stripTime(normalized.Date)
	if expenseDate.IsZero() {
		return nil, domain.NewDomainError(domain.ErrInvalidDate, "date is required", 400)
	}
	if expenseDate.After(dateOnly) {
		return nil, domain.NewDomainError(domain.ErrInvalidDate, "date cannot be in the future", 400)
	}

	currency := strings.ToUpper(strings.TrimSpace(normalized.Currency))
	if currency == "" {
		currency = "USD"
	}
	if len(currency) != 3 {
		return nil, domain.NewDomainError(domain.ErrInvalidCurrency, "currency must be a 3-letter ISO code", 400)
	}
	normalized.Currency = currency

	if normalized.FXRateToBase <= 0 {
		normalized.FXRateToBase = 1
	}
	normalized.FXRateToBase = round2(normalized.FXRateToBase)

	if normalized.Merchant != nil {
		trimmed := strings.TrimSpace(*normalized.Merchant)
		if trimmed == "" {
			normalized.Merchant = nil
		} else {
			normalized.Merchant = &trimmed
		}
	}
	if normalized.Notes != nil {
		trimmed := strings.TrimSpace(*normalized.Notes)
		if trimmed == "" {
			normalized.Notes = nil
		} else {
			normalized.Notes = &trimmed
		}
	}
	if normalized.RecurringRule != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*normalized.RecurringRule))
		if trimmed == "" {
			normalized.RecurringRule = nil
		} else if trimmed != "daily" && trimmed != "weekly" && trimmed != "monthly" && trimmed != "yearly" {
			return nil, domain.NewDomainError(domain.ErrInvalidDate, "recurring_rule must be daily, weekly, monthly, or yearly", 400)
		} else {
			normalized.RecurringRule = &trimmed
		}
	}

	normalized.Date = expenseDate
	return &normalized, nil
}

func stripTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
