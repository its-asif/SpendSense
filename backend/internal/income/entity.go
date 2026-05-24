package income

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Income struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	WalletID   uuid.UUID
	CategoryID *uuid.UUID
	SourceName string
	Amount     float64
	Currency   string
	IncomeDate time.Time
	Notes      *string
	IsDeleted  bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

type CreateRequest struct {
	WalletID   uuid.UUID
	CategoryID *uuid.UUID
	SourceName string
	Amount     float64
	Currency   string
	IncomeDate time.Time
	Notes      *string
}

type UpdateRequest struct {
	WalletID   uuid.UUID
	CategoryID *uuid.UUID
	SourceName string
	Amount     float64
	Currency   string
	IncomeDate time.Time
	Notes      *string
}

type Cursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
}

func (c Cursor) Encode() string {
	if c.CreatedAt.IsZero() || c.ID == uuid.Nil {
		return ""
	}

	data, err := json.Marshal(c)
	if err != nil {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString(data)
}

func DecodeCursor(value string) (*Cursor, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	decoded, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, err
	}

	var cursor Cursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, err
	}

	if cursor.CreatedAt.IsZero() || cursor.ID == uuid.Nil {
		return nil, fmt.Errorf("invalid cursor")
	}

	return &cursor, nil
}
