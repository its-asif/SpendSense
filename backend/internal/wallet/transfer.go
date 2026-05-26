package wallet

import (
	"time"

	"github.com/google/uuid"
)

type Transfer struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	FromWalletID    uuid.UUID
	ToWalletID      uuid.UUID
	Amount          float64
	ConvertedAmount float64
	ExchangeRate    float64
	FeeAmount       float64
	Currency        string
	TransferDate    time.Time
	Notes           *string
	CreatedAt       time.Time
}

type CreateTransferRequest struct {
	FromWalletID uuid.UUID `json:"from_wallet_id"`
	ToWalletID   uuid.UUID `json:"to_wallet_id"`
	Amount       float64   `json:"amount"`
	ExchangeRate float64   `json:"exchange_rate,omitempty"`
	FeeAmount    float64   `json:"fee_amount"`
	Currency     string    `json:"currency"`
	TransferDate string    `json:"transfer_date"`
	Notes        *string   `json:"notes,omitempty"`
}
