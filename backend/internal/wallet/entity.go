package wallet

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Name           string
	WalletType     string
	Provider       *string
	AccountNumber  *string
	AccountName    *string
	Currency       string
	OpeningBalance float64
	CurrentBalance float64
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateRequest struct {
	Name           string
	WalletType     string
	Provider       *string
	AccountNumber  *string
	AccountName    *string
	Currency       string
	OpeningBalance float64
}

type UpdateRequest struct {
	Name           string
	WalletType     string
	Provider       *string
	AccountNumber  *string
	AccountName    *string
	Currency       string
	IsActive       bool
	CurrentBalance float64
}
