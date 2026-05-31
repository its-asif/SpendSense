package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

type apiErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("api request failed: %s", e.Message)
	}
	return fmt.Sprintf("api request failed (%s): %s", e.Code, e.Message)
}

type AuthUser struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	BaseCurrency string `json:"base_currency"`
	Timezone     string `json:"timezone"`
	Locale       string `json:"locale"`
}

type AuthResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         AuthUser `json:"user"`
}

type CurrentUserResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type Category struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Icon      *string `json:"icon,omitempty"`
	Color     *string `json:"color,omitempty"`
	IsDefault bool    `json:"is_default"`
	CreatedAt string  `json:"created_at"`
}

type CategoryListResponse struct {
	Categories []Category `json:"categories"`
}

type Wallet struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	WalletType     string  `json:"wallet_type"`
	Provider       *string `json:"provider,omitempty"`
	AccountNumber  *string `json:"account_number,omitempty"`
	AccountName    *string `json:"account_name,omitempty"`
	Currency       string  `json:"currency"`
	OpeningBalance float64 `json:"opening_balance"`
	CurrentBalance float64 `json:"current_balance"`
	IsActive       bool    `json:"is_active"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type WalletListResponse struct {
	Wallets []Wallet `json:"wallets"`
}

type Expense struct {
	ID            string  `json:"id"`
	WalletID      string  `json:"wallet_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	FXRateToBase  float64 `json:"fx_rate_to_base"`
	CategoryID    string  `json:"category_id"`
	Merchant      *string `json:"merchant,omitempty"`
	Date          string  `json:"date"`
	Notes         *string `json:"notes,omitempty"`
	IsRecurring   bool    `json:"is_recurring"`
	RecurringRule *string `json:"recurring_rule,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type ExpenseListResponse struct {
	Expenses       []Expense `json:"expenses"`
	NextPagination string    `json:"next_pagination"`
}

type CreateExpenseRequest struct {
	WalletID      string  `json:"wallet_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	FXRateToBase  float64 `json:"fx_rate_to_base"`
	CategoryID    string  `json:"category_id"`
	Merchant      *string `json:"merchant,omitempty"`
	Date          string  `json:"date"`
	Notes         *string `json:"notes,omitempty"`
	IsRecurring   bool    `json:"is_recurring"`
	RecurringRule *string `json:"recurring_rule,omitempty"`
}

type ExpenseResponse = Expense

// Income types and client methods
type Income struct {
	ID         string  `json:"id"`
	WalletID   string  `json:"wallet_id"`
	CategoryID *string `json:"category_id,omitempty"`
	SourceName string  `json:"source_name"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	IncomeDate string  `json:"income_date"`
	Notes      *string `json:"notes,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type IncomeListResponse struct {
	Incomes        []Income `json:"incomes"`
	NextPagination string   `json:"next_pagination"`
}

type CreateIncomeRequest struct {
	WalletID   string  `json:"wallet_id"`
	CategoryID *string `json:"category_id,omitempty"`
	SourceName string  `json:"source_name"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	IncomeDate string  `json:"income_date"`
	Notes      *string `json:"notes,omitempty"`
}

type IncomeResponse = Income

func (c *APIClient) ListIncomes(ctx context.Context, limit int, pagination, from, to, categoryID string) (*IncomeListResponse, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if pagination != "" {
		query.Set("pagination", pagination)
	}
	if from != "" {
		query.Set("from", from)
	}
	if to != "" {
		query.Set("to", to)
	}
	if categoryID != "" {
		query.Set("category_id", categoryID)
	}

	var resp IncomeListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/incomes?"+query.Encode(), nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) CreateIncome(ctx context.Context, req CreateIncomeRequest) (*IncomeResponse, error) {
	var resp IncomeResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/incomes", req, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) DeleteIncome(ctx context.Context, incomeID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/incomes/"+url.PathEscape(incomeID), nil, nil, true)
}

func newAPIClient() *APIClient {
	baseURL := strings.TrimRight(viper.GetString("api_url"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &APIClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 20 * time.Second},
		token:      strings.TrimSpace(viper.GetString("access_token")),
	}
}

func clientOSHeaderValue() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}

func clientUserAgent() string {
	return fmt.Sprintf("spendsense-cli/1.0 (%s; %s)", clientOSHeaderValue(), runtime.GOARCH)
}

func (c *APIClient) Register(ctx context.Context, email, password string) (*AuthResponse, error) {
	var resp AuthResponse
	if err := c.doJSON(ctx, http.MethodPost, "/auth/register", map[string]string{"email": email, "password": password}, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) Login(ctx context.Context, email, password string, totpCode string) (*AuthResponse, error) {
	var resp AuthResponse
	body := map[string]string{"email": email, "password": password}
	if strings.TrimSpace(totpCode) != "" {
		body["totp_code"] = strings.TrimSpace(totpCode)
	}
	if err := c.doJSON(ctx, http.MethodPost, "/auth/login", body, &resp, false); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) Refresh(ctx context.Context, email, refreshToken string) (string, error) {
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/auth/refresh", map[string]string{"email": email, "refresh_token": refreshToken}, &resp, false); err != nil {
		return "", err
	}
	return resp.AccessToken, nil
}

func (c *APIClient) Logout(ctx context.Context, refreshToken string) error {
	return c.doJSON(ctx, http.MethodPost, "/auth/logout", map[string]string{"refresh_token": refreshToken}, nil, true)
}

func (c *APIClient) LogoutAll(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodPost, "/auth/logout-all", nil, nil, true)
}

func (c *APIClient) Me(ctx context.Context) (*CurrentUserResponse, error) {
	var resp CurrentUserResponse
	if err := c.doJSON(ctx, http.MethodGet, "/auth/me", nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) ListCategories(ctx context.Context) (*CategoryListResponse, error) {
	var resp CategoryListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/categories", nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) ListWallets(ctx context.Context) (*WalletListResponse, error) {
	var resp WalletListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/wallets", nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) ListExpenses(ctx context.Context, limit int, pagination, from, to, categoryID string) (*ExpenseListResponse, error) {
	query := url.Values{}
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if pagination != "" {
		query.Set("pagination", pagination)
	}
	if from != "" {
		query.Set("from", from)
	}
	if to != "" {
		query.Set("to", to)
	}
	if categoryID != "" {
		query.Set("category_id", categoryID)
	}

	var resp ExpenseListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/expenses?"+query.Encode(), nil, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) CreateExpense(ctx context.Context, req CreateExpenseRequest) (*ExpenseResponse, error) {
	var resp ExpenseResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/expenses", req, &resp, true); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *APIClient) DeleteExpense(ctx context.Context, expenseID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/expenses/"+url.PathEscape(expenseID), nil, nil, true)
}

func (c *APIClient) doJSON(ctx context.Context, method, path string, body any, out any, authRequired bool) error {
	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Type", "cli")
	req.Header.Set("X-Client-OS", clientOSHeaderValue())
	req.Header.Set("X-Platform", clientOSHeaderValue())
	req.Header.Set("User-Agent", clientUserAgent())
	if authRequired || c.token != "" {
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp)
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func decodeAPIError(resp *http.Response) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	apiErr := &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(data))}
	if len(data) > 0 {
		var parsed apiErrorResponse
		if err := json.Unmarshal(data, &parsed); err == nil && parsed.Error.Message != "" {
			apiErr.Code = parsed.Error.Code
			apiErr.Message = parsed.Error.Message
		}
	}
	if apiErr.Message == "" {
		apiErr.Message = resp.Status
	}
	return apiErr
}
