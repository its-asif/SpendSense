export type AuthUser = {
  name: string;
  email: string;
  baseCurrency: string;
};

export type ApiUser = {
  id: string;
  email: string;
  display_name?: string | null;
  avatar_url?: string | null;
  base_currency: string;
  timezone: string;
  locale: string;
  created_at: string;
  updated_at: string;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type LoginRequest = {
  email: string;
  password: string;
  totp_code?: string;
};

export type RegisterRequest = {
  email: string;
  password: string;
};

export type ApiAuthResponse = {
  access_token: string;
  refresh_token: string;
  user: ApiUser;
};

export type ApiTokenResponse = {
  access_token: string;
};

export type ApiCurrentUser = {
  user_id: string;
  email: string;
};

export type ApiSessionRow = {
  session_id: string;
  id?: string;
  device: string;
  ip: string;
  last_seen: string;
  refresh_token_hash: string;
  revoked: boolean;
  created_at: string;
  expires_at: string;
};

export type ApiListSessionsResponse = {
  sessions: ApiSessionRow[];
};

export type ExpenseCategory = {
  id: string;
  name: string;
  icon?: string | null;
  color?: string | null;
  is_default: boolean;
  created_at: string;
};

export type Wallet = {
  id: string;
  name: string;
  wallet_type: string;
  provider?: string | null;
  account_number?: string | null;
  account_name?: string | null;
  currency: string;
  opening_balance: number;
  current_balance: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type Expense = {
  id: string;
  wallet_id: string;
  amount: number;
  currency: string;
  fx_rate_to_base: number;
  category_id: string;
  merchant?: string | null;
  date: string;
  notes?: string | null;
  is_recurring: boolean;
  recurring_rule?: string | null;
  created_at: string;
  updated_at: string;
};

export type Income = {
  id: string;
  wallet_id: string;
  amount: number;
  currency: string;
  category_id?: string | null;
  source_name: string;
  income_date: string;
  notes?: string | null;
  created_at: string;
  updated_at: string;
  deleted_at?: string | null;
};

export type CreateExpenseRequest = {
  wallet_id: string;
  amount: number;
  currency: string;
  fx_rate_to_base?: number;
  category_id: string;
  merchant?: string;
  date: string;
  notes?: string;
  is_recurring?: boolean;
  recurring_rule?: string;
};

export type UpdateExpenseRequest = CreateExpenseRequest;

export type CreateIncomeRequest = {
  wallet_id: string;
  amount: number;
  currency: string;
  category_id?: string;
  source_name: string;
  income_date: string;
  notes?: string;
};

export type UpdateIncomeRequest = CreateIncomeRequest;

export type ListExpensesResponse = {
  expenses: Expense[];
  next_pagination?: string | null;
};

export type ListIncomesResponse = {
  incomes: Income[];
  next_pagination?: string | null;
};

export type CategoryListResponse = {
  categories: ExpenseCategory[];
};

export type CreateCategoryRequest = {
  name: string;
  icon?: string;
  color?: string;
};

export type UpdateCategoryRequest = CreateCategoryRequest;

export type WalletListResponse = {
  wallets: Wallet[];
};

export type CreateWalletRequest = {
  name: string;
  wallet_type: string;
  provider?: string;
  account_number?: string;
  account_name?: string;
  currency: string;
  opening_balance: number;
};

export type UpdateWalletRequest = CreateWalletRequest;

export type CurrencyOption = {
  code: string;
  name: string;
  symbol: string;
  symbol_native: string;
  decimal_digits: number;
  rounding: number;
  name_plural: string;
  is_default: boolean;
};

export type CurrencyListResponse = {
  default_currency: string;
  currencies: CurrencyOption[];
};

export type CurrencyConvertResponse = {
  from_currency: string;
  to_currency: string;
  amount: number;
  converted_amount: number;
  exchange_rate: number;
};

export type TransactionType = 'income' | 'expense';

export type Transaction = {
  id: string;
  category: string;
  description: string;
  amount: number;
  type: TransactionType;
  date: string;
};

export type DashboardDailySpending = {
  date: string;
  label: string;
  total: number;
};

export type DashboardCategoryTotal = {
  category_id: string;
  name: string;
  total: number;
};

export type DashboardBudgetRow = {
  id: string;
  category_id: string;
  category_name: string;
  category_icon?: string | null;
  category_color?: string | null;
  limit: number;
  spent: number;
  usage_percent: number;
  period: string;
  currency: string;
};

export type DashboardMonthlyCashFlowPoint = {
  month: string;
  label: string;
  income: number;
  expenses: number;
};

export type DashboardSummary = {
  base_currency: string;
  total_balance: number;
  monthly_income: number;
  monthly_expenses: number;
  monthly_savings: number;
  net_this_month: number;
  safe_to_spend: number;
  monthly_spending_percent_change?: number | null;
  daily_spending: DashboardDailySpending[];
  category_breakdown: DashboardCategoryTotal[];
};

export type DashboardWidgetsResponse = {
  base_currency: string;
  budgets: DashboardBudgetRow[];
  monthly_cash_flow: DashboardMonthlyCashFlowPoint[];
};
