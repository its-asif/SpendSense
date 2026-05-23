package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/expense"
	"spendsense-backend/internal/middleware"

	"github.com/google/uuid"
)

type createExpenseRequest struct {
	WalletID      string  `json:"wallet_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	FXRateToBase  float64 `json:"fx_rate_to_base"`
	CategoryID    string  `json:"category_id"`
	Merchant      *string `json:"merchant,omitempty"`
	Date          string  `json:"date"` // YYYY-MM-DD
	Notes         *string `json:"notes,omitempty"`
	IsRecurring   bool    `json:"is_recurring,omitempty"`
	RecurringRule *string `json:"recurring_rule,omitempty"`
}

type expenseResponse struct {
	ID            uuid.UUID `json:"id"`
	WalletID      uuid.UUID `json:"wallet_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	FXRateToBase  float64   `json:"fx_rate_to_base"`
	CategoryID    uuid.UUID `json:"category_id"`
	Merchant      *string   `json:"merchant,omitempty"`
	Date          string    `json:"date"`
	Notes         *string   `json:"notes,omitempty"`
	IsRecurring   bool      `json:"is_recurring"`
	RecurringRule *string   `json:"recurring_rule,omitempty"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
}

func (s *Server) registerExpenseRoutes() {
	s.mux.Handle("/api/v1/expenses", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleCreateListExpenses)))
	s.mux.Handle("/api/v1/expenses/", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleExpenseByID)))
}

func (s *Server) handleCreateListExpenses(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req createExpenseRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}

		// map to service CreateRequest
		walletUUID, _ := uuid.Parse(req.WalletID)
		catUUID, _ := uuid.Parse(req.CategoryID)

		parsedDate, _ := time.Parse("2006-01-02", req.Date)

		svcReq := expense.CreateRequest{
			WalletID:      walletUUID,
			Amount:        req.Amount,
			Currency:      req.Currency,
			FXRateToBase:  req.FXRateToBase,
			CategoryID:    catUUID,
			Merchant:      req.Merchant,
			Date:          parsedDate,
			Notes:         req.Notes,
			IsRecurring:   req.IsRecurring,
			RecurringRule: req.RecurringRule,
		}

		created, err := s.expenseService.CreateExpense(r.Context(), userID, svcReq)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, toExpenseResponse(created))
		return
	case http.MethodGet:
		q := r.URL.Query()
		limit := 20
		if l := q.Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil {
				limit = v
			}
		}
		cursor := q.Get("cursor")

		list, next, err := s.expenseService.ListExpenses(r.Context(), userID, limit, cursor)
		if err != nil {
			writeError(w, err)
			return
		}

		resp := make([]expenseResponse, 0, len(list))
		for _, e := range list {
			resp = append(resp, toExpenseResponse(e))
		}

		writeJSON(w, http.StatusOK, map[string]any{"expenses": resp, "next_cursor": next})
		return
	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
}

func (s *Server) handleExpenseByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/expenses/")
	idStr = strings.Trim(idStr, "/")
	expenseID, err := uuid.Parse(idStr)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_EXPENSE_ID", "Invalid expense id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		e, err := s.expenseService.GetExpense(r.Context(), userID, expenseID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toExpenseResponse(e))
		return
	case http.MethodPut:
		var req createExpenseRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}
		walletUUID, _ := uuid.Parse(req.WalletID)
		catUUID, _ := uuid.Parse(req.CategoryID)
		parsedDate, _ := time.Parse("2006-01-02", req.Date)
		svcReq := expense.UpdateRequest{
			WalletID:      walletUUID,
			Amount:        req.Amount,
			Currency:      req.Currency,
			FXRateToBase:  req.FXRateToBase,
			CategoryID:    catUUID,
			Merchant:      req.Merchant,
			Date:          parsedDate,
			Notes:         req.Notes,
			IsRecurring:   req.IsRecurring,
			RecurringRule: req.RecurringRule,
		}

		updated, err := s.expenseService.UpdateExpense(r.Context(), userID, expenseID, svcReq)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toExpenseResponse(updated))
		return
	case http.MethodDelete:
		if err := s.expenseService.SoftDeleteExpense(r.Context(), userID, expenseID); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusNoContent, nil)
		return
	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
}

func toExpenseResponse(e *expense.Expense) expenseResponse {
	var dateStr string
	if !e.Date.IsZero() {
		dateStr = e.Date.Format("2006-01-02")
	}
	return expenseResponse{
		ID:            e.ID,
		WalletID:      e.WalletID,
		Amount:        e.Amount,
		Currency:      e.Currency,
		FXRateToBase:  e.FXRateToBase,
		CategoryID:    e.CategoryID,
		Merchant:      e.Merchant,
		Date:          dateStr,
		Notes:         e.Notes,
		IsRecurring:   e.IsRecurring,
		RecurringRule: e.RecurringRule,
		CreatedAt:     e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     e.UpdatedAt.Format(time.RFC3339),
	}
}
