package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"spendsense-backend/internal/infra"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var budgetThresholds = []int{75, 90, 100}

type Store interface {
	Create(ctx context.Context, n *Notification) error
	List(ctx context.Context, userID uuid.UUID, limit int, unreadOnly bool) ([]*Notification, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	MarkRead(ctx context.Context, userID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	Dismiss(ctx context.Context, userID, id uuid.UUID) error
	BudgetAlertAlreadySent(ctx context.Context, budgetID uuid.UUID, threshold int, yearMonth string) (bool, error)
	RecordBudgetAlertSent(ctx context.Context, budgetID uuid.UUID, threshold int, yearMonth string) error
	ClearBudgetAlertSent(ctx context.Context, budgetID uuid.UUID, threshold int, yearMonth string) error
}

type DatabaseReader interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Service struct {
	repo Store
	db   DatabaseReader
	now  func() time.Time
}

func NewService(db *infra.Database) *Service {
	return &Service{repo: NewRepository(db), db: db, now: time.Now}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, limit int, unreadOnly bool) ([]*Notification, error) {
	return s.repo.List(ctx, userID, limit, unreadOnly)
}

func (s *Service) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnread(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) Dismiss(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Dismiss(ctx, userID, id)
}

// RunChecks evaluates budget, recurring expense, and loan reminders for a user.
func (s *Service) RunChecks(ctx context.Context, userID uuid.UUID) error {
	if err := s.checkBudgetAlerts(ctx, userID); err != nil {
		log.Printf("budget alert check failed for user %s: %v", userID, err)
	}
	if err := s.checkRecurringExpenseReminders(ctx, userID); err != nil {
		log.Printf("recurring reminder check failed for user %s: %v", userID, err)
	}
	if err := s.checkLoanReminders(ctx, userID); err != nil {
		log.Printf("loan reminder check failed for user %s: %v", userID, err)
	}
	return nil
}

func (s *Service) checkBudgetAlerts(ctx context.Context, userID uuid.UUID) error {
	now := s.now().UTC()
	yearMonth := now.Format("2006-01")
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	rows, err := s.db.Query(ctx, `
		SELECT
			b.id,
			b.amount,
			b.currency,
			COALESCE(c.id::text, ''),
			COALESCE(c.name, 'Uncategorized'),
			COALESCE(SUM(e.amount), 0) AS spent
		FROM budgets b
		LEFT JOIN categories c ON c.id = b.category_id
		LEFT JOIN expenses e ON e.category_id = b.category_id
			AND e.user_id = b.user_id
			AND e.is_deleted = FALSE
			AND e.date >= $2 AND e.date < $3
		WHERE b.user_id = $1 AND UPPER(b.period) = 'MONTHLY'
		GROUP BY b.id, b.amount, b.currency, c.id, c.name
	`, userID, monthStart, monthEnd)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var budgetID uuid.UUID
		var limitAmount, spent float64
		var currency, categoryID, categoryName string
		if err := rows.Scan(&budgetID, &limitAmount, &currency, &categoryID, &categoryName, &spent); err != nil {
			return err
		}
		if limitAmount <= 0 {
			continue
		}

		usagePercent := int(math.Floor((spent / limitAmount) * 100))
		for _, threshold := range budgetThresholds {
			if usagePercent < threshold {
				if err := s.repo.ClearBudgetAlertSent(ctx, budgetID, threshold, yearMonth); err != nil {
					log.Printf("failed to clear budget alert for budget %s threshold %d: %v", budgetID, threshold, err)
				}
				continue
			}
			sent, err := s.repo.BudgetAlertAlreadySent(ctx, budgetID, threshold, yearMonth)
			if err != nil || sent {
				continue
			}

			title, body := budgetAlertCopy(threshold, categoryName, usagePercent, spent, limitAmount, currency)
			metadata, _ := json.Marshal(map[string]any{
				"budget_id":        budgetID.String(),
				"category_id":      categoryID,
				"category_name":    categoryName,
				"threshold_percent": threshold,
				"usage_percent":    usagePercent,
				"year_month":       yearMonth,
			})

			n := &Notification{
				ID:       uuid.New(),
				UserID:   userID,
				Type:     TypeBudgetAlert,
				Title:    title,
				Body:     body,
				Metadata: metadata,
			}
			if err := s.repo.Create(ctx, n); err != nil {
				return err
			}
			if err := s.repo.RecordBudgetAlertSent(ctx, budgetID, threshold, yearMonth); err != nil {
				return err
			}
		}
	}
	return rows.Err()
}

func budgetAlertCopy(threshold int, categoryName string, usagePercent int, spent, limit float64, currency string) (string, string) {
	switch threshold {
	case 100:
		return fmt.Sprintf("%s budget exceeded", categoryName),
			fmt.Sprintf("You have spent %.2f %s (%.0f%%) against your %.2f %s monthly limit.", spent, currency, float64(usagePercent), limit, currency)
	case 90:
		return fmt.Sprintf("%s budget at %d%%", categoryName, threshold),
			fmt.Sprintf("Spending has reached %d%% of your %.2f %s monthly budget for %s.", usagePercent, limit, currency, categoryName)
	default:
		return fmt.Sprintf("%s budget at %d%%", categoryName, threshold),
			fmt.Sprintf("You have used %d%% of your %.2f %s monthly budget for %s.", usagePercent, limit, currency, categoryName)
	}
}

func (s *Service) checkRecurringExpenseReminders(ctx context.Context, userID uuid.UUID) error {
	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	rows, err := s.db.Query(ctx, `
		SELECT id, title, amount, currency, start_date, deadline, alert_rule, interval
		FROM recurring_payments
		WHERE user_id = $1 AND status = 'unpaid'
	`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rpID uuid.UUID
		var title, currency, alertRule, interval string
		var amount float64
		var startDate, deadline time.Time
		if err := rows.Scan(&rpID, &title, &amount, &currency, &startDate, &deadline, &alertRule, &interval); err != nil {
			return err
		}

		// Determine if this should trigger an alert
		var triggerAlert bool
		if alertRule == "start" {
			triggerAlert = !today.Before(startDate)
		} else if strings.HasSuffix(alertRule, "d") {
			var days int
			if _, err := fmt.Sscanf(alertRule, "%dd", &days); err == nil {
				triggerAlert = !today.Before(deadline.AddDate(0, 0, -days))
			} else {
				triggerAlert = !today.Before(startDate)
			}
		} else {
			triggerAlert = !today.Before(startDate)
		}

		if !triggerAlert {
			continue
		}

		dedupeKey := fmt.Sprintf("rp:%s:%s", rpID.String(), deadline.Format("2006-01-02"))

		// Check if we already sent notification for this cycle
		var marker int
		errExists := s.db.QueryRow(ctx, `
			SELECT 1 FROM notifications
			WHERE user_id = $1 AND type = $2 AND metadata->>'dedupe_key' = $3
			LIMIT 1
		`, userID, TypeRecurringDue, dedupeKey).Scan(&marker)

		if errExists == nil {
			// Already notified
			continue
		}

		metadata, _ := json.Marshal(map[string]any{
			"recurring_payment_id": rpID.String(),
			"dedupe_key":           dedupeKey,
		})

		n := &Notification{
			ID:       uuid.New(),
			UserID:   userID,
			Type:     TypeRecurringDue,
			Title:    "Recurring payment due: " + title,
			Body:     fmt.Sprintf("Payment for %s of %.2f %s is due by %s.", title, amount, currency, deadline.Format("Jan 02, 2006")),
			Metadata: metadata,
		}
		if err := s.repo.Create(ctx, n); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Service) checkLoanReminders(ctx context.Context, userID uuid.UUID) error {
	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	horizon := today.AddDate(0, 0, 7)

	rows, err := s.db.Query(ctx, `
		SELECT id, counterparty_name, loan_direction, principal_amount, currency, due_date
		FROM personal_loans
		WHERE user_id = $1
			AND status IN ('ACTIVE', 'PARTIAL', 'OVERDUE')
			AND due_date IS NOT NULL
			AND due_date >= $2
			AND due_date <= $3
			AND (reminder_at IS NULL OR reminder_at <= $4)
			AND (snooze_until IS NULL OR snooze_until <= $4)
	`, userID, today, horizon, now)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var loanID uuid.UUID
		var counterparty, direction, currency string
		var principal float64
		var dueDate time.Time
		if err := rows.Scan(&loanID, &counterparty, &direction, &principal, &currency, &dueDate); err != nil {
			return err
		}

		dedupeKey := fmt.Sprintf("loan:%s:%s", loanID.String(), dueDate.Format("2006-01-02"))
		exists, err := s.notificationExistsToday(ctx, userID, TypeLoanReminder, dedupeKey)
		if err != nil || exists {
			continue
		}

		directionLabel := "borrowed from"
		if strings.EqualFold(direction, "LENT") {
			directionLabel = "lent to"
		}
		daysUntil := int(dueDate.Sub(today).Hours() / 24)

		metadata, _ := json.Marshal(map[string]any{
			"loan_id":    loanID.String(),
			"dedupe_key": dedupeKey,
			"due_date":   dueDate.Format("2006-01-02"),
		})
		n := &Notification{
			ID:     uuid.New(),
			UserID: userID,
			Type:   TypeLoanReminder,
			Title:  "Loan payment reminder",
			Body:   fmt.Sprintf("%.2f %s %s %s is due %s (%d day(s)).", principal, currency, directionLabel, counterparty, dueDate.Format("Jan 2, 2006"), daysUntil),
			Metadata: metadata,
		}
		if err := s.repo.Create(ctx, n); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Service) notificationExistsToday(ctx context.Context, userID uuid.UUID, notifType, dedupeKey string) (bool, error) {
	row := s.db.QueryRow(ctx, `
		SELECT 1 FROM notifications
		WHERE user_id = $1
			AND type = $2
			AND metadata->>'dedupe_key' = $3
			AND created_at::date = CURRENT_DATE
		LIMIT 1
	`, userID, notifType, dedupeKey)
	var marker int
	if err := row.Scan(&marker); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
