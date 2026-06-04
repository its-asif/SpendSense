package notification

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TypeBudgetAlert    = "budget_alert"
	TypeRecurringDue   = "recurring_due"
	TypeLoanReminder   = "loan_reminder"
)

type Notification struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Type        string
	Title       string
	Body        string
	Metadata    json.RawMessage
	IsRead      bool
	CreatedAt   time.Time
	DismissedAt *time.Time
}
