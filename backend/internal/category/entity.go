package category

import (
	"time"

	"github.com/google/uuid"
)

const (
	KindExpense = "EXPENSE"
	KindIncome  = "INCOME"
)

type Category struct {
	ID        uuid.UUID
	UserID    *uuid.UUID
	Name      string
	Icon      *string
	Color     *string
	Kind      string
	IsDefault bool
	CreatedAt time.Time
}

type CreateRequest struct {
	Name  string
	Icon  *string
	Color *string
	Kind  string
}

type UpdateRequest struct {
	Name  string
	Icon  *string
	Color *string
}
