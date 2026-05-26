package wallet

import (
	"context"
	"math"
	"strings"
	"time"

	"spendsense-backend/internal/currency"
	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type RepositoryInterface interface {
	CreateWallet(ctx context.Context, w *Wallet) error
	GetWalletByID(ctx context.Context, userID, id uuid.UUID) (*Wallet, error)
	ListWallets(ctx context.Context, userID uuid.UUID) ([]*Wallet, error)
	UpdateWallet(ctx context.Context, w *Wallet) error
	DeleteWallet(ctx context.Context, userID, id uuid.UUID) error
	CreateTransfer(ctx context.Context, t *Transfer) error
}

type Service struct {
	repo        RepositoryInterface
	currencySvc *currency.Service
}

func NewService(r RepositoryInterface, currencySvc *currency.Service) *Service {
	return &Service{repo: r, currencySvc: currencySvc}
}

func (s *Service) CreateWallet(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Wallet, error) {
	if req.Name == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidWallet, "name is required", 400)
	}

	currencyCode := req.Currency
	if s.currencySvc != nil {
		resolved, err := s.currencySvc.NormalizeOrDefault(currencyCode, req.WalletType, req.Provider)
		if err != nil {
			return nil, domain.NewDomainError(domain.ErrInvalidCurrency, err.Error(), 400)
		}
		currencyCode = resolved
	} else {
		currencyCode = roundCurrencyCode(currencyCode)
	}

	w := &Wallet{
		ID:             uuid.New(),
		UserID:         userID,
		Name:           req.Name,
		WalletType:     req.WalletType,
		Provider:       req.Provider,
		AccountNumber:  req.AccountNumber,
		AccountName:    req.AccountName,
		Currency:       currencyCode,
		OpeningBalance: round2(req.OpeningBalance),
		CurrentBalance: round2(req.OpeningBalance),
		IsActive:       true,
		CreatedAt:      time.Time{},
		UpdatedAt:      time.Time{},
	}

	if err := s.repo.CreateWallet(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) GetWallet(ctx context.Context, userID, id uuid.UUID) (*Wallet, error) {
	return s.repo.GetWalletByID(ctx, userID, id)
}

func (s *Service) ListWallets(ctx context.Context, userID uuid.UUID) ([]*Wallet, error) {
	return s.repo.ListWallets(ctx, userID)
}

func (s *Service) UpdateWallet(ctx context.Context, userID, id uuid.UUID, req UpdateRequest) (*Wallet, error) {
	w, err := s.repo.GetWalletByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	w.Name = req.Name
	w.WalletType = req.WalletType
	w.Provider = req.Provider
	w.AccountNumber = req.AccountNumber
	w.AccountName = req.AccountName
	if s.currencySvc != nil {
		resolved, err := s.currencySvc.NormalizeOrDefault(req.Currency, req.WalletType, req.Provider)
		if err != nil {
			return nil, domain.NewDomainError(domain.ErrInvalidCurrency, err.Error(), 400)
		}
		w.Currency = resolved
	} else if req.Currency != "" {
		w.Currency = roundCurrencyCode(req.Currency)
	}
	w.IsActive = req.IsActive
	w.CurrentBalance = round2(req.CurrentBalance)

	if err := s.repo.UpdateWallet(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) DeleteWallet(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.DeleteWallet(ctx, userID, id)
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func roundCurrencyCode(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(strings.TrimSpace(value))
}
