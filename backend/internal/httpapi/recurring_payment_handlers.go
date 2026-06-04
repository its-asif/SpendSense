package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"spendsense-backend/internal/expense"
	"spendsense-backend/internal/middleware"

	"github.com/google/uuid"
)

const recurringDateFormat = "2006-01-02"

type recurringPaymentRequest struct {
	WalletID   string   `json:"wallet_id"`
	CategoryID string   `json:"category_id"`
	Title      string   `json:"title"`
	Amount     float64  `json:"amount"`
	Currency   string   `json:"currency"`
	Interval   string   `json:"interval"`   // daily, weekly, monthly, yearly
	StartDate  string   `json:"start_date"` // YYYY-MM-DD
	Deadline   string   `json:"deadline"`   // YYYY-MM-DD
	AlertRule  string   `json:"alert_rule"` // start, 1d, 7d, 1h, 12h
	EndDate    *string  `json:"end_date,omitempty"`
}

type recurringPaymentResponse struct {
	ID            string  `json:"id"`
	WalletID      string  `json:"wallet_id"`
	WalletName    string  `json:"wallet_name"`
	CategoryID    string  `json:"category_id"`
	CategoryName  string  `json:"category_name"`
	CategoryColor string  `json:"category_color"`
	CategoryIcon  string  `json:"category_icon"`
	Title         string  `json:"title"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Interval      string  `json:"interval"`
	StartDate     string  `json:"start_date"`
	Deadline      string  `json:"deadline"`
	AlertRule     string  `json:"alert_rule"`
	EndDate       *string `json:"end_date,omitempty"`
	Status        string  `json:"status"` // unpaid, paid, inactive
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type payRecurringRequest struct {
	PaymentDate string  `json:"payment_date"`
	Fine        float64 `json:"fine"`
}

func (s *Server) registerRecurringPaymentRoutes() {
	s.mux.Handle("/api/v1/recurring-payments", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleCreateListRecurringPayments)))
	s.mux.Handle("/api/v1/recurring-payments/", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleRecurringPaymentByID)))
}

func (s *Server) handleCreateListRecurringPayments(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req recurringPaymentRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}

		walletUUID, err := uuid.Parse(req.WalletID)
		if err != nil {
			writeStatusError(w, http.StatusBadRequest, "INVALID_WALLET_ID", "Invalid wallet id")
			return
		}
		categoryUUID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			writeStatusError(w, http.StatusBadRequest, "INVALID_CATEGORY_ID", "Invalid category id")
			return
		}

		startDate, err := time.Parse(recurringDateFormat, req.StartDate)
		if err != nil {
			writeStatusError(w, http.StatusBadRequest, "INVALID_START_DATE", "Start date must be YYYY-MM-DD")
			return
		}

		deadline, err := time.Parse(recurringDateFormat, req.Deadline)
		if err != nil {
			writeStatusError(w, http.StatusBadRequest, "INVALID_DEADLINE", "Deadline must be YYYY-MM-DD")
			return
		}

		var endDate *time.Time
		if req.EndDate != nil && strings.TrimSpace(*req.EndDate) != "" {
			parsedEnd, err := time.Parse(recurringDateFormat, *req.EndDate)
			if err != nil {
				writeStatusError(w, http.StatusBadRequest, "INVALID_END_DATE", "End date must be YYYY-MM-DD")
				return
			}
			endDate = &parsedEnd
		}

		var id uuid.UUID
		err = s.db.QueryRow(r.Context(), `
			INSERT INTO recurring_payments (user_id, wallet_id, category_id, title, amount, currency, interval, start_date, deadline, alert_rule, end_date, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'unpaid')
			RETURNING id
		`, userID, walletUUID, categoryUUID, req.Title, req.Amount, req.Currency, req.Interval, startDate, deadline, req.AlertRule, endDate).Scan(&id)
		if err != nil {
			writeError(w, err)
			return
		}

		resp, err := s.fetchRecurringPaymentResponseByID(r.Context(), userID, id)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, resp)
		return

	case http.MethodGet:
		rows, err := s.db.Query(r.Context(), `
			SELECT rp.id, rp.wallet_id, w.name, rp.category_id, c.name, COALESCE(c.color, ''), COALESCE(c.icon, ''),
			       rp.title, rp.amount, rp.currency, rp.interval, rp.start_date, rp.deadline, rp.alert_rule, rp.end_date, rp.status, rp.created_at, rp.updated_at
			FROM recurring_payments rp
			LEFT JOIN wallets w ON rp.wallet_id = w.id
			LEFT JOIN categories c ON rp.category_id = c.id
			WHERE rp.user_id = $1
			ORDER BY rp.deadline ASC
		`, userID)
		if err != nil {
			writeError(w, err)
			return
		}
		defer rows.Close()

		list := []recurringPaymentResponse{}
		for rows.Next() {
			var id, wID, cID uuid.UUID
			var wName, cName, cColor, cIcon, title, currency, interval, alertRule, status string
			var amount float64
			var startDate, deadline, createdAt, updatedAt time.Time
			var endDate *time.Time

			err := rows.Scan(&id, &wID, &wName, &cID, &cName, &cColor, &cIcon, &title, &amount, &currency, &interval, &startDate, &deadline, &alertRule, &endDate, &status, &createdAt, &updatedAt)
			if err != nil {
				writeError(w, err)
				return
			}

			var endDateStr *string
			if endDate != nil {
				s := endDate.Format(recurringDateFormat)
				endDateStr = &s
			}

			list = append(list, recurringPaymentResponse{
				ID:            id.String(),
				WalletID:      wID.String(),
				WalletName:    wName,
				CategoryID:    cID.String(),
				CategoryName:  cName,
				CategoryColor: cColor,
				CategoryIcon:  cIcon,
				Title:         title,
				Amount:        amount,
				Currency:      currency,
				Interval:      interval,
				StartDate:     startDate.Format(recurringDateFormat),
				Deadline:      deadline.Format(recurringDateFormat),
				AlertRule:     alertRule,
				EndDate:       endDateStr,
				Status:        status,
				CreatedAt:     createdAt.Format(time.RFC3339),
				UpdatedAt:     updatedAt.Format(time.RFC3339),
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{"recurring_payments": list})
		return

	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
}

func (s *Server) handleRecurringPaymentByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/recurring-payments/")
	idStr = strings.Trim(idStr, "/")

	// If there's a "/pay" suffix
	isPayAction := false
	if strings.HasSuffix(idStr, "/pay") {
		isPayAction = true
		idStr = strings.TrimSuffix(idStr, "/pay")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_ID", "Invalid recurring payment id")
		return
	}

	if isPayAction {
		if r.Method != http.MethodPost {
			writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
			return
		}
		s.handlePayRecurringPayment(w, r, userID, id)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req recurringPaymentRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}

		walletUUID, err := uuid.Parse(req.WalletID)
		if err != nil {
			writeStatusError(w, http.StatusBadRequest, "INVALID_WALLET_ID", "Invalid wallet id")
			return
		}
		categoryUUID, err := uuid.Parse(req.CategoryID)
		if err != nil {
			writeStatusError(w, http.StatusBadRequest, "INVALID_CATEGORY_ID", "Invalid category id")
			return
		}

		startDate, err := time.Parse(recurringDateFormat, req.StartDate)
		if err != nil {
			writeStatusError(w, http.StatusBadRequest, "INVALID_START_DATE", "Start date must be YYYY-MM-DD")
			return
		}

		deadline, err := time.Parse(recurringDateFormat, req.Deadline)
		if err != nil {
			writeStatusError(w, http.StatusBadRequest, "INVALID_DEADLINE", "Deadline must be YYYY-MM-DD")
			return
		}

		var endDate *time.Time
		if req.EndDate != nil && strings.TrimSpace(*req.EndDate) != "" {
			parsedEnd, err := time.Parse(recurringDateFormat, *req.EndDate)
			if err != nil {
				writeStatusError(w, http.StatusBadRequest, "INVALID_END_DATE", "End date must be YYYY-MM-DD")
				return
			}
			endDate = &parsedEnd
		}

		tag, err := s.db.Exec(r.Context(), `
			UPDATE recurring_payments
			SET wallet_id = $1, category_id = $2, title = $3, amount = $4, currency = $5, interval = $6, start_date = $7, deadline = $8, alert_rule = $9, end_date = $10, updated_at = CURRENT_TIMESTAMP
			WHERE id = $11 AND user_id = $12
		`, walletUUID, categoryUUID, req.Title, req.Amount, req.Currency, req.Interval, startDate, deadline, req.AlertRule, endDate, id, userID)
		if err != nil {
			writeError(w, err)
			return
		}

		if tag.RowsAffected() == 0 {
			writeStatusError(w, http.StatusNotFound, "NOT_FOUND", "Recurring payment not found")
			return
		}

		resp, err := s.fetchRecurringPaymentResponseByID(r.Context(), userID, id)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, resp)
		return

	case http.MethodDelete:
		tag, err := s.db.Exec(r.Context(), `
			DELETE FROM recurring_payments
			WHERE id = $1 AND user_id = $2
		`, id, userID)
		if err != nil {
			writeError(w, err)
			return
		}

		if tag.RowsAffected() == 0 {
			writeStatusError(w, http.StatusNotFound, "NOT_FOUND", "Recurring payment not found")
			return
		}

		writeJSON(w, http.StatusNoContent, nil)
		return

	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
}

func (s *Server) handlePayRecurringPayment(w http.ResponseWriter, r *http.Request, userID, id uuid.UUID) {
	var req payRecurringRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	paymentDate, err := time.Parse(recurringDateFormat, req.PaymentDate)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_PAYMENT_DATE", "Payment date must be YYYY-MM-DD")
		return
	}

	// Fetch current recurring payment state
	var walletID, categoryID uuid.UUID
	var title, currency, interval, alertRule, status string
	var amount float64
	var startDate, deadline time.Time
	var endDate *time.Time

	err = s.db.QueryRow(r.Context(), `
		SELECT wallet_id, category_id, title, amount, currency, interval, start_date, deadline, alert_rule, end_date, status
		FROM recurring_payments
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&walletID, &categoryID, &title, &amount, &currency, &interval, &startDate, &deadline, &alertRule, &endDate, &status)
	if err != nil {
		writeStatusError(w, http.StatusNotFound, "NOT_FOUND", "Recurring payment not found")
		return
	}

	// Double check fine input if paid after deadline
	actualAmount := amount
	if req.Fine > 0 {
		actualAmount += req.Fine
	}

	notes := "Recurring payment cycle: " + startDate.Format(recurringDateFormat) + " to " + deadline.Format(recurringDateFormat)
	if req.Fine > 0 {
		notes += fmt.Sprintf(" (includes fine/penalty of %.2f)", req.Fine)
	}

	// Post the transaction as a regular expense using the expense service!
	// This automatically handles currency logic, updates wallet balances, and triggers budget alerts!
	svcReq := expense.CreateRequest{
		WalletID:      walletID,
		Amount:        actualAmount,
		Currency:      currency,
		FXRateToBase:  1.0,
		CategoryID:    categoryID,
		Merchant:      &title,
		Date:          paymentDate,
		Notes:         &notes,
		IsRecurring:   false,
		RecurringRule: nil,
	}

	_, err = s.expenseService.CreateExpense(r.Context(), userID, svcReq)
	if err != nil {
		writeError(w, err)
		return
	}

	// Advance dates by the interval
	var nextStart, nextDeadline time.Time
	switch strings.ToLower(interval) {
	case "daily":
		nextStart = startDate.AddDate(0, 0, 1)
		nextDeadline = deadline.AddDate(0, 0, 1)
	case "weekly":
		nextStart = startDate.AddDate(0, 0, 7)
		nextDeadline = deadline.AddDate(0, 0, 7)
	case "yearly":
		nextStart = startDate.AddDate(1, 0, 0)
		nextDeadline = deadline.AddDate(1, 0, 0)
	case "monthly":
		fallthrough
	default:
		nextStart = startDate.AddDate(0, 1, 0)
		nextDeadline = deadline.AddDate(0, 1, 0)
	}

	nextStatus := "unpaid"
	if endDate != nil && (nextStart.After(*endDate) || nextStart.Equal(*endDate)) {
		nextStatus = "inactive"
	}

	// Save advanced recurring payment cycle
	_, err = s.db.Exec(r.Context(), `
		UPDATE recurring_payments
		SET start_date = $1, deadline = $2, status = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`, nextStart, nextDeadline, nextStatus, id)
	if err != nil {
		writeError(w, err)
		return
	}

	resp, err := s.fetchRecurringPaymentResponseByID(r.Context(), userID, id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) fetchRecurringPaymentResponseByID(ctx context.Context, userID, id uuid.UUID) (recurringPaymentResponse, error) {
	var wID, cID uuid.UUID
	var wName, cName, cColor, cIcon, title, currency, interval, alertRule, status string
	var amount float64
	var startDate, deadline, createdAt, updatedAt time.Time
	var endDate *time.Time

	err := s.db.QueryRow(ctx, `
		SELECT rp.id, rp.wallet_id, w.name, rp.category_id, c.name, COALESCE(c.color, ''), COALESCE(c.icon, ''),
		       rp.title, rp.amount, rp.currency, rp.interval, rp.start_date, rp.deadline, rp.alert_rule, rp.end_date, rp.status, rp.created_at, rp.updated_at
		FROM recurring_payments rp
		LEFT JOIN wallets w ON rp.wallet_id = w.id
		LEFT JOIN categories c ON rp.category_id = c.id
		WHERE rp.id = $1 AND rp.user_id = $2
	`, id, userID).Scan(&id, &wID, &wName, &cID, &cName, &cColor, &cIcon, &title, &amount, &currency, &interval, &startDate, &deadline, &alertRule, &endDate, &status, &createdAt, &updatedAt)
	if err != nil {
		return recurringPaymentResponse{}, err
	}

	var endDateStr *string
	if endDate != nil {
		s := endDate.Format(recurringDateFormat)
		endDateStr = &s
	}

	return recurringPaymentResponse{
		ID:            id.String(),
		WalletID:      wID.String(),
		WalletName:    wName,
		CategoryID:    cID.String(),
		CategoryName:  cName,
		CategoryColor: cColor,
		CategoryIcon:  cIcon,
		Title:         title,
		Amount:        amount,
		Currency:      currency,
		Interval:      interval,
		StartDate:     startDate.Format(recurringDateFormat),
		Deadline:      deadline.Format(recurringDateFormat),
		AlertRule:     alertRule,
		EndDate:       endDateStr,
		Status:        status,
		CreatedAt:     createdAt.Format(time.RFC3339),
		UpdatedAt:     updatedAt.Format(time.RFC3339),
	}, nil
}
