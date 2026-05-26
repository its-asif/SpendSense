package report

type DailySpending struct {
	Date  string  `json:"date"`
	Label string  `json:"label"`
	Total float64 `json:"total"`
}

type CategoryTotal struct {
	CategoryID string  `json:"category_id"`
	Name       string  `json:"name"`
	Total      float64 `json:"total"`
}

type BudgetOverview struct {
	ID            string  `json:"id"`
	CategoryID    string  `json:"category_id"`
	CategoryName  string  `json:"category_name"`
	CategoryIcon  *string `json:"category_icon,omitempty"`
	CategoryColor *string `json:"category_color,omitempty"`
	Limit         float64 `json:"limit"`
	Spent         float64 `json:"spent"`
	UsagePercent  float64 `json:"usage_percent"`
	Period        string  `json:"period"`
	Currency      string  `json:"currency"`
}

type MonthlyCashFlowPoint struct {
	Month    string  `json:"month"`
	Label    string  `json:"label"`
	Income   float64 `json:"income"`
	Expenses float64 `json:"expenses"`
}

type DashboardSummary struct {
	BaseCurrency                 string          `json:"base_currency"`
	TotalBalance                 float64         `json:"total_balance"`
	MonthlyIncome                float64         `json:"monthly_income"`
	MonthlyExpenses              float64         `json:"monthly_expenses"`
	MonthlySavings               float64         `json:"monthly_savings"`
	NetThisMonth                 float64         `json:"net_this_month"`
	SafeToSpend                  float64         `json:"safe_to_spend"`
	MonthlySpendingPercentChange *float64        `json:"monthly_spending_percent_change,omitempty"`
	DailySpending                []DailySpending `json:"daily_spending"`
	CategoryBreakdown            []CategoryTotal `json:"category_breakdown"`
}

type DashboardWidgets struct {
	BaseCurrency    string                 `json:"base_currency"`
	Budgets         []BudgetOverview       `json:"budgets"`
	MonthlyCashFlow []MonthlyCashFlowPoint `json:"monthly_cash_flow"`
}
