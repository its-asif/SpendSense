package notification

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type mockRows struct {
	pgx.Rows
	data [][]any
	idx  int
	err  error
}

func (m *mockRows) Next() bool {
	return m.idx < len(m.data)
}

func (m *mockRows) Scan(dest ...any) error {
	if m.idx >= len(m.data) {
		return fmt.Errorf("no more rows")
	}
	row := m.data[m.idx]
	m.idx++
	if len(dest) != len(row) {
		return fmt.Errorf("scan destination length mismatch: got %d, expected %d", len(dest), len(row))
	}
	for i, val := range row {
		if val == nil {
			switch d := dest[i].(type) {
			case *sql.NullString:
				d.Valid = false
			}
			continue
		}

		switch d := dest[i].(type) {
		case *float64:
			*d = val.(float64)
		case *string:
			*d = val.(string)
		case *time.Time:
			*d = val.(time.Time)
		case *uuid.UUID:
			*d = val.(uuid.UUID)
		case *sql.NullString:
			d.String = val.(string)
			d.Valid = true
		default:
			return fmt.Errorf("unsupported scan type %T", d)
		}
	}
	return nil
}

func (m *mockRows) Close() {}

func (m *mockRows) Err() error {
	return m.err
}

type mockRow struct {
	val any
	err error
}

func (m *mockRow) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}
	if len(dest) > 0 {
		switch d := dest[0].(type) {
		case *int:
			*d = m.val.(int)
		case *string:
			*d = m.val.(string)
		}
	}
	return nil
}

type mockDB struct {
	budgetRows    [][]any
	recurringRows [][]any
	loanRows      [][]any
	rowExistsVal  int
}

func (m *mockDB) Query(ctx context.Context, sqlQuery string, args ...any) (pgx.Rows, error) {
	lowerSQL := strings.ToLower(sqlQuery)
	if strings.Contains(lowerSQL, "budgets") {
		return &mockRows{data: m.budgetRows}, nil
	}
	if strings.Contains(lowerSQL, "recurring_payments") {
		return &mockRows{data: m.recurringRows}, nil
	}
	if strings.Contains(lowerSQL, "personal_loans") {
		return &mockRows{data: m.loanRows}, nil
	}
	return &mockRows{}, nil
}

func (m *mockDB) QueryRow(ctx context.Context, sqlQuery string, args ...any) pgx.Row {
	return &mockRow{val: m.rowExistsVal}
}

type fakeNotificationStore struct {
	created          *Notification
	listVal          []*Notification
	countVal         int
	readID           uuid.UUID
	readUID          uuid.UUID
	allReadUID       uuid.UUID
	dismissID        uuid.UUID
	dismissUID       uuid.UUID
	budgetAlertSent  bool
	recordAlertSent  bool
	clearAlertSent   bool
}

func (f *fakeNotificationStore) Create(ctx context.Context, n *Notification) error {
	f.created = n
	return nil
}

func (f *fakeNotificationStore) List(ctx context.Context, userID uuid.UUID, limit int, unreadOnly bool) ([]*Notification, error) {
	return f.listVal, nil
}

func (f *fakeNotificationStore) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	return f.countVal, nil
}

func (f *fakeNotificationStore) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	f.readUID = userID
	f.readID = id
	return nil
}

func (f *fakeNotificationStore) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	f.allReadUID = userID
	return nil
}

func (f *fakeNotificationStore) Dismiss(ctx context.Context, userID, id uuid.UUID) error {
	f.dismissUID = userID
	f.dismissID = id
	return nil
}

func (f *fakeNotificationStore) BudgetAlertAlreadySent(ctx context.Context, budgetID uuid.UUID, threshold int, yearMonth string) (bool, error) {
	return f.budgetAlertSent, nil
}

func (f *fakeNotificationStore) RecordBudgetAlertSent(ctx context.Context, budgetID uuid.UUID, threshold int, yearMonth string) error {
	f.recordAlertSent = true
	return nil
}

func (f *fakeNotificationStore) ClearBudgetAlertSent(ctx context.Context, budgetID uuid.UUID, threshold int, yearMonth string) error {
	f.clearAlertSent = true
	return nil
}

func TestNotificationListAndCount(t *testing.T) {
	repo := &fakeNotificationStore{
		listVal: []*Notification{
			{ID: uuid.New(), Title: "Alert 1"},
		},
		countVal: 5,
	}
	svc := &Service{repo: repo}
	userID := uuid.New()

	// List
	list, err := svc.List(context.Background(), userID, 10, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].Title != "Alert 1" {
		t.Errorf("unexpected list output")
	}

	// Count
	count, err := svc.CountUnread(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}
}

func TestNotificationActions(t *testing.T) {
	repo := &fakeNotificationStore{}
	svc := &Service{repo: repo}
	userID := uuid.New()
	notifID := uuid.New()

	// Mark Read
	err := svc.MarkRead(context.Background(), userID, notifID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.readUID != userID || repo.readID != notifID {
		t.Errorf("repo.MarkRead called with unexpected parameters")
	}

	// Mark All Read
	err = svc.MarkAllRead(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.allReadUID != userID {
		t.Errorf("repo.MarkAllRead called with unexpected parameters")
	}

	// Dismiss
	err = svc.Dismiss(context.Background(), userID, notifID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.dismissUID != userID || repo.dismissID != notifID {
		t.Errorf("repo.Dismiss called with unexpected parameters")
	}
}

func TestRunChecks(t *testing.T) {
	repo := &fakeNotificationStore{}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	alertRule := "7d"
	db := &mockDB{
		budgetRows: [][]any{
			// budgetID, amount, currency, categoryID, categoryName, spent
			// 80/100 → 80% → triggers 75% threshold
			{uuid.New(), 100.0, "USD", uuid.New().String(), "Dining", 80.0},
		},
		recurringRows: [][]any{
			// rpID, title, amount, currency, startDate, deadline, alertRule, interval
			{uuid.New(), "Netflix", 15.0, "USD", today, today.AddDate(0, 0, 3), alertRule, "monthly"},
		},
		loanRows: [][]any{
			// loanID, counterparty, direction, principal, currency, dueDate
			{uuid.New(), "Charlie", "LENT", 200.0, "USD", today.AddDate(0, 0, 2)},
		},
		rowExistsVal: 0,
	}

	svc := &Service{
		repo: repo,
		db:   db,
		now:  func() time.Time { return now },
	}

	err := svc.RunChecks(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error running checks: %v", err)
	}

	if !repo.recordAlertSent {
		t.Errorf("expected budget alert record to be stored")
	}
	if repo.created == nil {
		t.Fatalf("expected a notification to be created")
	}
}

func TestRunChecksAlreadySent(t *testing.T) {
	// When alert already sent, no new notification should be created
	repo := &fakeNotificationStore{budgetAlertSent: true}
	now := time.Now().UTC()
	db := &mockDB{
		budgetRows: [][]any{
			{uuid.New(), 100.0, "USD", uuid.New().String(), "Food", 90.0},
		},
	}
	svc := &Service{repo: repo, db: db, now: func() time.Time { return now }}
	if err := svc.RunChecks(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.created != nil {
		t.Errorf("expected no notification when alert already sent")
	}
}

func TestRunChecksBudgetBelowThreshold(t *testing.T) {
	// Budget usage below 75% → clear alert sent, no notification
	repo := &fakeNotificationStore{}
	now := time.Now().UTC()
	db := &mockDB{
		budgetRows: [][]any{
			{uuid.New(), 100.0, "USD", uuid.New().String(), "Food", 50.0},
		},
	}
	svc := &Service{repo: repo, db: db, now: func() time.Time { return now }}
	if err := svc.RunChecks(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.created != nil {
		t.Errorf("expected no notification for budget below threshold")
	}
	if !repo.clearAlertSent {
		t.Errorf("expected ClearBudgetAlertSent to be called")
	}
}

func TestRunChecksRecurringAlreadyNotified(t *testing.T) {
	// rowExistsVal = 1 means notification already exists
	repo := &fakeNotificationStore{}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	db := &mockDB{
		recurringRows: [][]any{
			{uuid.New(), "Netflix", 15.0, "USD", today, today.AddDate(0, 0, 1), "1d", "monthly"},
		},
		rowExistsVal: 1, // already notified
	}
	svc := &Service{repo: repo, db: db, now: func() time.Time { return now }}
	if err := svc.RunChecks(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBudgetAlertCopy(t *testing.T) {
	tests := []struct {
		threshold int
		wantTitle string
	}{
		{100, "Food budget exceeded"},
		{90, "Food budget at 90%"},
		{75, "Food budget at 75%"},
	}
	for _, tc := range tests {
		title, body := budgetAlertCopy(tc.threshold, "Food", tc.threshold, 90, 100, "USD")
		if title != tc.wantTitle {
			t.Errorf("threshold %d: expected title %q, got %q", tc.threshold, tc.wantTitle, title)
		}
		if body == "" {
			t.Errorf("threshold %d: expected non-empty body", tc.threshold)
		}
	}
}

func TestRunChecksLoanReminders(t *testing.T) {
	repo := &fakeNotificationStore{}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	t.Run("BorrowedDirection", func(t *testing.T) {
		db := &mockDB{
			loanRows: [][]any{
				// loanID, counterparty, direction, principal, currency, dueDate
				{uuid.New(), "Alice", "BORROWED", 500.0, "USD", today.AddDate(0, 0, 3)},
			},
			rowExistsVal: 0,
		}
		svc := &Service{repo: repo, db: db, now: func() time.Time { return now }}
		if err := svc.RunChecks(context.Background(), uuid.New()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.created == nil || !strings.Contains(repo.created.Body, "borrowed from Alice") {
			t.Errorf("expected borrowed loan notification, got %+v", repo.created)
		}
	})

	t.Run("LoanAlreadyNotified", func(t *testing.T) {
		repo.created = nil
		db := &mockDB{
			loanRows: [][]any{
				{uuid.New(), "Alice", "BORROWED", 500.0, "USD", today.AddDate(0, 0, 3)},
			},
			rowExistsVal: 1, // already exists
		}
		svc := &Service{repo: repo, db: db, now: func() time.Time { return now }}
		if err := svc.RunChecks(context.Background(), uuid.New()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.created != nil {
			t.Errorf("expected no new notification since it already exists")
		}
	})
}

func TestRunChecksRecurringAlertRules(t *testing.T) {
	repo := &fakeNotificationStore{}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	t.Run("AlertRuleStart", func(t *testing.T) {
		db := &mockDB{
			recurringRows: [][]any{
				{uuid.New(), "Netflix", 15.0, "USD", today, today.AddDate(0, 0, 3), "start", "monthly"},
			},
		}
		svc := &Service{repo: repo, db: db, now: func() time.Time { return now }}
		if err := svc.RunChecks(context.Background(), uuid.New()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.created == nil {
			t.Errorf("expected recurring notification for 'start' alert rule")
		}
	})

	t.Run("AlertRuleInvalidDaysFormat", func(t *testing.T) {
		repo.created = nil
		db := &mockDB{
			recurringRows: [][]any{
				{uuid.New(), "Netflix", 15.0, "USD", today, today.AddDate(0, 0, 3), "invalid_rule", "monthly"},
			},
		}
		svc := &Service{repo: repo, db: db, now: func() time.Time { return now }}
		if err := svc.RunChecks(context.Background(), uuid.New()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.created == nil {
			t.Errorf("expected recurring notification fallback for invalid rule")
		}
	})
}

