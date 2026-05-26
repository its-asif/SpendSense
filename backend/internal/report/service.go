package report

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"spendsense-backend/internal/infra"

	"github.com/google/uuid"
)

type Converter interface {
	Convert(ctx context.Context, amount float64, fromCurrency, toCurrency string) (float64, float64, error)
}

type Service struct {
	db        *infra.Database
	converter Converter
}

func NewService(db *infra.Database, converter Converter) *Service {
	return &Service{db: db, converter: converter}
}

func (s *Service) DashboardSummary(ctx context.Context, userID uuid.UUID) (*DashboardSummary, error) {
	return s.dashboardSummary(ctx, userID, "")
}

func (s *Service) DashboardSummaryForCurrency(ctx context.Context, userID uuid.UUID, currencyOverride string) (*DashboardSummary, error) {
	return s.dashboardSummary(ctx, userID, currencyOverride)
}

func (s *Service) DashboardWidgetsForCurrency(ctx context.Context, userID uuid.UUID, currencyOverride string) (*DashboardWidgets, error) {
	return s.dashboardWidgets(ctx, userID, currencyOverride)
}

func (s *Service) dashboardSummary(ctx context.Context, userID uuid.UUID, currencyOverride string) (*DashboardSummary, error) {
	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	baseCurrency := strings.ToUpper(strings.TrimSpace(currencyOverride))
	if baseCurrency == "" {
		baseCurrency = strings.ToUpper(strings.TrimSpace(user.BaseCurrency))
	}
	if baseCurrency == "" {
		baseCurrency = "USD"
	}

	location := time.UTC
	if user.Timezone != "" {
		if loaded, err := time.LoadLocation(user.Timezone); err == nil {
			location = loaded
		}
	}

	now := time.Now().In(location)
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
	nextMonthStart := currentMonthStart.AddDate(0, 1, 0)
	lastMonthStart := currentMonthStart.AddDate(0, -1, 0)

	wallets, err := s.fetchWallets(ctx, userID)
	if err != nil {
		return nil, err
	}

	totalBalance, err := s.sumWallets(ctx, wallets, baseCurrency)
	fmt.Println(totalBalance)
	if err != nil {
		return nil, err
	}

	monthlyExpenses, expenseBreakdown, err := s.sumExpenses(ctx, userID, currentMonthStart, nextMonthStart, baseCurrency)
	if err != nil {
		return nil, err
	}

	lastMonthlyExpenses, _, err := s.sumExpenses(ctx, userID, lastMonthStart, currentMonthStart, baseCurrency)
	if err != nil {
		return nil, err
	}

	monthlyIncome, err := s.sumIncomes(ctx, userID, currentMonthStart, nextMonthStart, baseCurrency)
	if err != nil {
		return nil, err
	}

	dailySpending, err := s.dailySpending(ctx, userID, now, location, baseCurrency)
	if err != nil {
		return nil, err
	}

	var monthlyChange *float64
	if lastMonthlyExpenses > 0 {
		value := ((monthlyExpenses - lastMonthlyExpenses) / lastMonthlyExpenses) * 100
		monthlyChange = &value
	}

	netThisMonth := round2(monthlyIncome - monthlyExpenses)
	safeToSpend := round2(totalBalance)
	monthlySavings := round2(math.Max(netThisMonth, 0))

	return &DashboardSummary{
		BaseCurrency:                 baseCurrency,
		TotalBalance:                 round2(totalBalance),
		MonthlyIncome:                round2(monthlyIncome),
		MonthlyExpenses:              round2(monthlyExpenses),
		MonthlySavings:               monthlySavings,
		NetThisMonth:                 netThisMonth,
		SafeToSpend:                  safeToSpend,
		MonthlySpendingPercentChange: monthlyChange,
		DailySpending:                dailySpending,
		CategoryBreakdown:            expenseBreakdown,
	}, nil
}

func (s *Service) dashboardWidgets(ctx context.Context, userID uuid.UUID, currencyOverride string) (*DashboardWidgets, error) {
	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	baseCurrency := strings.ToUpper(strings.TrimSpace(currencyOverride))
	if baseCurrency == "" {
		baseCurrency = strings.ToUpper(strings.TrimSpace(user.BaseCurrency))
	}
	if baseCurrency == "" {
		baseCurrency = "USD"
	}

	location := time.UTC
	if user.Timezone != "" {
		if loaded, err := time.LoadLocation(user.Timezone); err == nil {
			location = loaded
		}
	}

	now := time.Now().In(location)
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
	nextMonthStart := currentMonthStart.AddDate(0, 1, 0)

	currentMonthTotal, categoryBreakdown, err := s.sumExpenses(ctx, userID, currentMonthStart, nextMonthStart, baseCurrency)
	if err != nil {
		return nil, err
	}
	_ = currentMonthTotal

	categoryTotals := make(map[string]float64, len(categoryBreakdown))
	for _, item := range categoryBreakdown {
		key := item.CategoryID
		if key == "" {
			key = item.Name
		}
		categoryTotals[key] = item.Total
	}

	type budgetRow struct {
		ID            string
		CategoryID    string
		CategoryName  string
		CategoryIcon  sql.NullString
		CategoryColor sql.NullString
		Amount        float64
		Currency      string
		Period        string
	}

	rows, err := s.db.Query(ctx, `
		SELECT b.id::text, COALESCE(c.id::text, ''), COALESCE(c.name, 'Uncategorized'), c.icon, c.color, b.amount, b.currency, b.period
		FROM budgets b
		LEFT JOIN categories c ON c.id = b.category_id
		WHERE b.user_id = $1 AND UPPER(b.period) = 'MONTHLY'
		ORDER BY COALESCE(c.name, 'Uncategorized') ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversionCache := map[string]float64{}
	budgets := make([]BudgetOverview, 0)
	for rows.Next() {
		var row budgetRow
		if err := rows.Scan(&row.ID, &row.CategoryID, &row.CategoryName, &row.CategoryIcon, &row.CategoryColor, &row.Amount, &row.Currency, &row.Period); err != nil {
			return nil, err
		}

		limit, err := s.convertAmount(ctx, row.Amount, row.Currency, baseCurrency, conversionCache)
		if err != nil {
			return nil, err
		}

		key := row.CategoryID
		if key == "" {
			key = row.CategoryName
		}
		spent := categoryTotals[key]
		usagePercent := 0.0
		if limit > 0 {
			usagePercent = round2((spent / limit) * 100)
		}

		budget := BudgetOverview{
			ID:           row.ID,
			CategoryID:   row.CategoryID,
			CategoryName: row.CategoryName,
			Limit:        round2(limit),
			Spent:        round2(spent),
			UsagePercent: usagePercent,
			Period:       strings.ToUpper(strings.TrimSpace(row.Period)),
			Currency:     baseCurrency,
		}
		if row.CategoryIcon.Valid {
			value := row.CategoryIcon.String
			budget.CategoryIcon = &value
		}
		if row.CategoryColor.Valid {
			value := row.CategoryColor.String
			budget.CategoryColor = &value
		}
		budgets = append(budgets, budget)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(budgets, func(i, j int) bool {
		if budgets[i].UsagePercent == budgets[j].UsagePercent {
			return budgets[i].CategoryName < budgets[j].CategoryName
		}
		return budgets[i].UsagePercent > budgets[j].UsagePercent
	})

	startMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location).AddDate(0, -9, 0)
	monthlyCashFlow := make([]MonthlyCashFlowPoint, 0, 10)
	for index := 0; index < 10; index++ {
		monthStart := startMonth.AddDate(0, index, 0)
		monthEnd := monthStart.AddDate(0, 1, 0)

		expenses, _, err := s.sumExpenses(ctx, userID, monthStart, monthEnd, baseCurrency)
		if err != nil {
			return nil, err
		}
		incomes, err := s.sumIncomes(ctx, userID, monthStart, monthEnd, baseCurrency)
		if err != nil {
			return nil, err
		}

		monthlyCashFlow = append(monthlyCashFlow, MonthlyCashFlowPoint{
			Month:    monthStart.Format("2006-01"),
			Label:    monthStart.Format("Jan 06"),
			Income:   round2(incomes),
			Expenses: round2(expenses),
		})
	}

	return &DashboardWidgets{
		BaseCurrency:    baseCurrency,
		Budgets:         budgets,
		MonthlyCashFlow: monthlyCashFlow,
	}, nil
}

type walletRow struct {
	Balance  float64
	Currency string
}

type expenseRow struct {
	Amount       float64
	Currency     string
	CategoryID   string
	CategoryName string
	Date         time.Time
}

type incomeRow struct {
	Amount   float64
	Currency string
	Date     time.Time
}

func (s *Service) fetchWallets(ctx context.Context, userID uuid.UUID) ([]walletRow, error) {
	rows, err := s.db.Query(ctx, `
		SELECT current_balance, currency
		FROM wallets
		WHERE user_id = $1 AND is_active = TRUE
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	wallets := make([]walletRow, 0)
	for rows.Next() {
		var row walletRow
		if err := rows.Scan(&row.Balance, &row.Currency); err != nil {
			return nil, err
		}
		wallets = append(wallets, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return wallets, nil
}

func (s *Service) sumWallets(ctx context.Context, wallets []walletRow, baseCurrency string) (float64, error) {
	var total float64
	conversionCache := map[string]float64{}
	for _, wallet := range wallets {
		converted, err := s.convertAmount(ctx, wallet.Balance, wallet.Currency, baseCurrency, conversionCache)
		if err != nil {
			return 0, err
		}
		total += converted
	}
	return total, nil
}

func (s *Service) sumExpenses(ctx context.Context, userID uuid.UUID, start, end time.Time, baseCurrency string) (float64, []CategoryTotal, error) {
	rows, err := s.db.Query(ctx, `
		SELECT e.amount, e.currency, COALESCE(c.id::text, ''), COALESCE(c.name, 'Uncategorized'), e.date
		FROM expenses e
		LEFT JOIN categories c ON c.id = e.category_id
		WHERE e.user_id = $1 AND e.is_deleted = FALSE AND e.date >= $2::date AND e.date < $3::date
		ORDER BY e.date ASC, e.created_at ASC
	`, userID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	conversionCache := map[string]float64{}
	categoryTotals := map[string]*CategoryTotal{}
	var total float64

	for rows.Next() {
		var row expenseRow
		if err := rows.Scan(&row.Amount, &row.Currency, &row.CategoryID, &row.CategoryName, &row.Date); err != nil {
			return 0, nil, err
		}

		converted, err := s.convertAmount(ctx, row.Amount, row.Currency, baseCurrency, conversionCache)
		if err != nil {
			return 0, nil, err
		}
		total += converted

		key := row.CategoryID
		if key == "" {
			key = row.CategoryName
		}
		current := categoryTotals[key]
		if current == nil {
			current = &CategoryTotal{CategoryID: row.CategoryID, Name: row.CategoryName}
			categoryTotals[key] = current
		}
		current.Total = round2(current.Total + converted)
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}

	breakdown := make([]CategoryTotal, 0, len(categoryTotals))
	for _, item := range categoryTotals {
		breakdown = append(breakdown, *item)
	}
	sort.Slice(breakdown, func(i, j int) bool {
		if breakdown[i].Total == breakdown[j].Total {
			return breakdown[i].Name < breakdown[j].Name
		}
		return breakdown[i].Total > breakdown[j].Total
	})

	return round2(total), breakdown, nil
}

func (s *Service) sumIncomes(ctx context.Context, userID uuid.UUID, start, end time.Time, baseCurrency string) (float64, error) {
	rows, err := s.db.Query(ctx, `
		SELECT amount, currency, income_date
		FROM incomes
		WHERE user_id = $1 AND is_deleted = FALSE AND income_date >= $2::date AND income_date < $3::date
		ORDER BY income_date ASC, created_at ASC
	`, userID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	conversionCache := map[string]float64{}
	var total float64
	for rows.Next() {
		var row incomeRow
		if err := rows.Scan(&row.Amount, &row.Currency, &row.Date); err != nil {
			return 0, err
		}

		converted, err := s.convertAmount(ctx, row.Amount, row.Currency, baseCurrency, conversionCache)
		if err != nil {
			return 0, err
		}
		total += converted
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	return round2(total), nil
}

func (s *Service) dailySpending(ctx context.Context, userID uuid.UUID, now time.Time, location *time.Location, baseCurrency string) ([]DailySpending, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	start := today.AddDate(0, 0, -6)
	end := today.AddDate(0, 0, 1)

	rows, err := s.db.Query(ctx, `
		SELECT amount, currency, date
		FROM expenses
		WHERE user_id = $1 AND is_deleted = FALSE AND date >= $2::date AND date < $3::date
		ORDER BY date ASC, created_at ASC
	`, userID, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversionCache := map[string]float64{}
	totals := make(map[string]float64, 7)
	for rows.Next() {
		var amount float64
		var currencyCode string
		var date time.Time
		if err := rows.Scan(&amount, &currencyCode, &date); err != nil {
			return nil, err
		}

		converted, err := s.convertAmount(ctx, amount, currencyCode, baseCurrency, conversionCache)
		if err != nil {
			return nil, err
		}
		dateKey := date.In(location).Format("2006-01-02")
		totals[dateKey] = round2(totals[dateKey] + converted)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	labels := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	result := make([]DailySpending, 0, 7)
	for i := 0; i < 7; i++ {
		current := start.AddDate(0, 0, i)
		key := current.Format("2006-01-02")
		result = append(result, DailySpending{
			Date:  key,
			Label: labels[int(current.Weekday())],
			Total: round2(totals[key]),
		})
	}

	return result, nil
}

func (s *Service) convertAmount(ctx context.Context, amount float64, fromCurrency, toCurrency string, cache map[string]float64) (float64, error) {
	fromCurrency = strings.ToUpper(strings.TrimSpace(fromCurrency))
	toCurrency = strings.ToUpper(strings.TrimSpace(toCurrency))
	if fromCurrency == "" || toCurrency == "" {
		return 0, fmt.Errorf("currency codes are required")
	}
	if fromCurrency == toCurrency {
		return round2(amount), nil
	}

	cacheKey := fromCurrency + "->" + toCurrency
	if rate, ok := cache[cacheKey]; ok {
		return round2(amount * rate), nil
	}

	if s.converter == nil {
		return 0, fmt.Errorf("currency converter is not configured")
	}

	converted, rate, err := s.converter.Convert(ctx, amount, fromCurrency, toCurrency)
	if err != nil {
		return 0, err
	}
	cache[cacheKey] = rate
	return round2(converted), nil
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
