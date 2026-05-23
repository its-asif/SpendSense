package wallet

import (
	"context"
	"time"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(r *Repository) *Service { return &Service{repo: r} }

func (s *Service) CreateWallet(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Wallet, error) {
	if req.Name == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidWallet, "name is required", 400)
	}

	w := &Wallet{
		ID:             uuid.New(),
		UserID:         userID,
		Name:           req.Name,
		WalletType:     req.WalletType,
		Provider:       req.Provider,
		AccountNumber:  req.AccountNumber,
		AccountName:    req.AccountName,
		Currency:       req.Currency,
		OpeningBalance: req.OpeningBalance,
		CurrentBalance: req.OpeningBalance,
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
	w.Currency = req.Currency
	w.IsActive = req.IsActive
	w.CurrentBalance = req.CurrentBalance

	if err := s.repo.UpdateWallet(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) DeleteWallet(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.DeleteWallet(ctx, userID, id)
}
