package expense

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Expense struct {
	ID            uuid.UUID
	UserID        uuid.UUID
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
	IsDeleted     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

type CreateRequest struct {
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

type UpdateRequest struct {
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

type Pagination struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

func (p Pagination) Encode() string {
	if p.CreatedAt.IsZero() || p.ID == uuid.Nil {
		return ""
	}

	data, err := json.Marshal(p)
	if err != nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(data)
}

func DecodePagination(value string) (*Pagination, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, err
	}

	var pagination Pagination
	if err := json.Unmarshal(decoded, &pagination); err != nil {
		return nil, err
	}

	if pagination.CreatedAt.IsZero() || pagination.ID == uuid.Nil {
		return nil, fmt.Errorf("invalid pagination")
	}

	return &pagination, nil
}
