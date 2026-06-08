package report

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"spendsense-backend/internal/domain"

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
			default:
				// Leave nil or keep unchanged
			}
			continue
		}

		switch d := dest[i].(type) {
		case *float64:
			if v, ok := val.(float64); ok {
				*d = v
			} else if v, ok := val.(int); ok {
				*d = float64(v)
			} else {
				return fmt.Errorf("type mismatch at field %d: expected float64, got %T", i, val)
			}
		case *string:
			*d = val.(string)
		case *time.Time:
			*d = val.(time.Time)
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

type mockDB struct {
	user          *domain.User
	userErr       error
	walletRows    [][]any
	expenseRows   [][]any
	incomeRows    [][]any
	recurringRows [][]any
	budgetRows    [][]any
}

func (m *mockDB) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	if m.userErr != nil {
		return nil, m.userErr
	}
	if m.user != nil {
		return m.user, nil
	}
	return &domain.User{
		ID:           userID,
		Email:        "user@example.com",
		BaseCurrency: "USD",
		Timezone:     "UTC",
	}, nil
}

func (m *mockDB) Query(ctx context.Context, sqlQuery string, args ...any) (pgx.Rows, error) {
	lowerSQL := strings.ToLower(sqlQuery)
	if strings.Contains(lowerSQL, "wallets") {
		return &mockRows{data: m.walletRows}, nil
	}
	if strings.Contains(lowerSQL, "expenses") {
		// Distinguish between sumExpenses (which requests category fields) and dailySpending (which only requests amount, currency, date)
		if strings.Contains(lowerSQL, "categories") || strings.Contains(lowerSQL, "coalesce") {
			return &mockRows{data: m.expenseRows}, nil
		}
		// For dailySpending, map expenseRows {amount, currency, category_id, category_name, date} to {amount, currency, date}
		daily := make([][]any, 0, len(m.expenseRows))
		for _, r := range m.expenseRows {
			if len(r) >= 5 {
				daily = append(daily, []any{r[0], r[1], r[4]})
			}
		}
		return &mockRows{data: daily}, nil
	}
	if strings.Contains(lowerSQL, "incomes") {
		return &mockRows{data: m.incomeRows}, nil
	}
	if strings.Contains(lowerSQL, "recurring_payments") {
		return &mockRows{data: m.recurringRows}, nil
	}
	if strings.Contains(lowerSQL, "budgets") {
		return &mockRows{data: m.budgetRows}, nil
	}
	return &mockRows{}, nil
}

type mockConverter struct {
	rate float64
	err  error
}

func (mc *mockConverter) Convert(ctx context.Context, amount float64, fromCurrency, toCurrency string) (float64, float64, error) {
	if mc.err != nil {
		return 0, 0, mc.err
	}
	rate := mc.rate
	if rate == 0 {
		rate = 1.0
	}
	return amount * rate, rate, nil
}

func TestDashboardSummary(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	catID := uuid.New().String()

	db := &mockDB{
		walletRows: [][]any{
			{500.0, "USD"},
			{100.0, "EUR"},
		},
		expenseRows: [][]any{
			{50.0, "USD", catID, "Food", now},
		},
		incomeRows: [][]any{
			{1000.0, "USD", now},
		},
		recurringRows: [][]any{
			{25.0, "USD"},
		},
	}

	converter := &mockConverter{rate: 1.0}
	svc := NewService(db, converter)

	summary, err := svc.DashboardSummary(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.BaseCurrency != "USD" {
		t.Errorf("expected USD base currency, got %s", summary.BaseCurrency)
	}
	// Wallets: 500 + 100 = 600
	if summary.TotalBalance != 600.0 {
		t.Errorf("expected 600 balance, got %f", summary.TotalBalance)
	}
	if summary.MonthlyIncome != 1000.0 {
		t.Errorf("expected 1000 income, got %f", summary.MonthlyIncome)
	}
	if summary.MonthlyExpenses != 50.0 {
		t.Errorf("expected 50 expenses, got %f", summary.MonthlyExpenses)
	}
	if summary.SafeToSpend != 575.0 { // TotalBalance(600) - UnpaidRecurring(25) = 575
		t.Errorf("expected 575 safe to spend, got %f", summary.SafeToSpend)
	}
}

func TestDashboardWidgets(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	catID := uuid.New().String()

	db := &mockDB{
		budgetRows: [][]any{
			{uuid.New().String(), catID, "Dining", "utensils", "red", 300.0, "USD", "MONTHLY"},
		},
		expenseRows: [][]any{
			{60.0, "USD", catID, "Dining", now},
		},
	}

	converter := &mockConverter{rate: 1.0}
	svc := NewService(db, converter)

	widgets, err := svc.DashboardWidgetsForCurrency(context.Background(), userID, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(widgets.Budgets) != 1 {
		t.Fatalf("expected 1 budget overview, got %d", len(widgets.Budgets))
	}

	bo := widgets.Budgets[0]
	if bo.CategoryName != "Dining" || bo.Limit != 300.0 {
		t.Errorf("unexpected budget details: %+v", bo)
	}
	// 60 / 300 = 20%
	if bo.UsagePercent != 20.0 {
		t.Errorf("expected 20%% usage, got %f", bo.UsagePercent)
	}
}
