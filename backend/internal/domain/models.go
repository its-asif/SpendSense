package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	DisplayName  *string
	AvatarURL    *string
	BaseCurrency string
	Timezone     string
	Locale       string
	// TOTP (2FA)
	TOTPSecret      *string
	TOTPEnabled     bool
	TOTPConfirmedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

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
	RecurringRule *string // "daily", "weekly", "monthly", "yearly"
	IsDeleted     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

type Category struct {
	ID        uuid.UUID
	UserID    *uuid.UUID
	Name      string
	Icon      *string
	Color     *string
	IsDefault bool
	CreatedAt time.Time
}

type AuditLog struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	EntityType string // "expense", "budget", "category"
	EntityID   uuid.UUID
	Action     string // "create", "update", "delete"
	Before     *string
	After      *string
	IPAddress  string
	CreatedAt  time.Time
}
