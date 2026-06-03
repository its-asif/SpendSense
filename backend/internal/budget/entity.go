package budget

import (
	"time"

	"github.com/google/uuid"
)

const (
	PeriodMonthly = "MONTHLY"
	PeriodWeekly  = "WEEKLY"
	PeriodYearly  = "YEARLY"
)

type Budget struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	CategoryID      uuid.UUID
	CategoryName    string
	CategoryIcon    *string
	CategoryColor   *string
	Amount          float64
	Currency        string
	Period          string
	StartDate       time.Time
	RolloverEnabled bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreateRequest struct {
	CategoryID      uuid.UUID
	Amount          float64
	Currency        string
	Period          string
	StartDate       time.Time
	RolloverEnabled bool
}

type UpdateRequest struct {
	CategoryID      uuid.UUID
	Amount          float64
	Currency        string
	Period          string
	StartDate       time.Time
	RolloverEnabled bool
}
