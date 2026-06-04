package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/expense"
	"spendsense-backend/internal/middleware"

	"github.com/google/uuid"
)

const expenseDateFormat = "2006-01-02"

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
	ReceiptURL    *string   `json:"receipt_url,omitempty"`
}

func (s *Server) registerExpenseRoutes() {
	s.mux.Handle("/api/v1/expenses", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleCreateListExpenses)))
	s.mux.Handle("/api/v1/expenses/recurring/post", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handlePostRecurringExpense)))
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
		s.scheduleNotificationChecks(userID)

		writeJSON(w, http.StatusCreated, s.enrichExpenseResponse(r.Context(), created))
		return
	case http.MethodGet:
		q := r.URL.Query()
		limit := 20
		if l := q.Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil {
				limit = v
			}
		}
		pagination := q.Get("pagination")

		from, err := parseOptionalExpenseDate(q.Get("from"))
		if err != nil {
			writeStatusError(w, http.StatusBadRequest, "INVALID_FROM_DATE", "from must use YYYY-MM-DD")
			return
		}
		to, err := parseOptionalExpenseDate(q.Get("to"))
		if err != nil {
			writeStatusError(w, http.StatusBadRequest, "INVALID_TO_DATE", "to must use YYYY-MM-DD")
			return
		}
		if from != nil && to != nil && from.After(*to) {
			writeStatusError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", "from must be on or before to")
			return
		}

		var categoryID *uuid.UUID
		if value := strings.TrimSpace(q.Get("category_id")); value != "" {
			parsed, err := uuid.Parse(value)
			if err != nil {
				writeStatusError(w, http.StatusBadRequest, "INVALID_CATEGORY_ID", "category_id must be a valid UUID")
				return
			}
			categoryID = &parsed
		}

		list, next, err := s.expenseService.ListExpenses(r.Context(), userID, limit, pagination, from, to, categoryID)
		if err != nil {
			writeError(w, err)
			return
		}

		resp := make([]expenseResponse, 0, len(list))
		for _, e := range list {
			resp = append(resp, s.enrichExpenseResponse(r.Context(), e))
		}

		writeJSON(w, http.StatusOK, map[string]any{"expenses": resp, "next_pagination": next})
		return
	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
}

func parseOptionalExpenseDate(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse(expenseDateFormat, trimmed)
	if err != nil {
		return nil, err
	}

	dateOnly := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	return &dateOnly, nil
}

func (s *Server) handleExpenseByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/expenses/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		writeStatusError(w, http.StatusBadRequest, "INVALID_EXPENSE_ID", "Invalid expense id")
		return
	}

	expenseID, err := uuid.Parse(parts[0])
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_EXPENSE_ID", "Invalid expense id")
		return
	}

	if len(parts) == 2 && parts[1] == "receipt" {
		s.handleUploadReceipt(w, r, userID, expenseID)
		return
	}

	if len(parts) > 1 {
		writeStatusError(w, http.StatusNotFound, "NOT_FOUND", "Not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		e, err := s.expenseService.GetExpense(r.Context(), userID, expenseID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, s.enrichExpenseResponse(r.Context(), e))
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
		s.scheduleNotificationChecks(userID)
		writeJSON(w, http.StatusOK, s.enrichExpenseResponse(r.Context(), updated))
		return
	case http.MethodDelete:
		if err := s.expenseService.SoftDeleteExpense(r.Context(), userID, expenseID); err != nil {
			writeError(w, err)
			return
		}
		s.scheduleNotificationChecks(userID)
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

func (s *Server) enrichExpenseResponse(ctx context.Context, e *expense.Expense) expenseResponse {
	resp := toExpenseResponse(e)

	var filePath string
	err := s.db.QueryRow(ctx, "SELECT file_path FROM receipts WHERE expense_id = $1 LIMIT 1", e.ID).Scan(&filePath)
	if err == nil && filePath != "" {
		filename := filePath
		if idx := strings.LastIndex(filePath, "/"); idx != -1 {
			filename = filePath[idx+1:]
		}
		url := "/assets/receipts/" + filename
		resp.ReceiptURL = &url
	}

	return resp
}

type postRecurringRequest struct {
	ExpenseID string  `json:"expense_id"`
	Date      *string `json:"date,omitempty"`
}

func (s *Server) handlePostRecurringExpense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	var req postRecurringRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	parsedExpenseID, err := uuid.Parse(req.ExpenseID)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_EXPENSE_ID", "Invalid expense id")
		return
	}

	tmpl, err := s.expenseService.GetExpense(r.Context(), userID, parsedExpenseID)
	if err != nil {
		writeError(w, err)
		return
	}

	if !tmpl.IsRecurring {
		writeStatusError(w, http.StatusBadRequest, "NOT_RECURRING", "Selected expense is not a recurring template")
		return
	}

	targetDate := time.Now().UTC()
	if req.Date != nil && *req.Date != "" {
		if d, err := time.Parse("2006-01-02", *req.Date); err == nil {
			targetDate = d
		} else {
			writeStatusError(w, http.StatusBadRequest, "INVALID_DATE", "Date must be in YYYY-MM-DD format")
			return
		}
	}

	svcReq := expense.CreateRequest{
		WalletID:     tmpl.WalletID,
		Amount:       tmpl.Amount,
		Currency:     tmpl.Currency,
		FXRateToBase: tmpl.FXRateToBase,
		CategoryID:   tmpl.CategoryID,
		Merchant:     tmpl.Merchant,
		Date:         targetDate,
		Notes:        tmpl.Notes,
		IsRecurring:  false,
	}

	posted, err := s.expenseService.CreateExpense(r.Context(), userID, svcReq)
	if err != nil {
		writeError(w, err)
		return
	}

	s.scheduleNotificationChecks(userID)

	writeJSON(w, http.StatusCreated, s.enrichExpenseResponse(r.Context(), posted))
}

func (s *Server) handleUploadReceipt(w http.ResponseWriter, r *http.Request, userID, expenseID uuid.UUID) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	e, err := s.expenseService.GetExpense(r.Context(), userID, expenseID)
	if err != nil {
		writeError(w, err)
		return
	}

	err = r.ParseMultipartForm(5 << 20)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "Maximum upload size is 5MB")
		return
	}

	file, header, err := r.FormFile("receipt")
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "MISSING_FILE", "multipart file field 'receipt' is required")
		return
	}
	defer file.Close()

	dir := "./assets/receipts"
	if err := os.MkdirAll(dir, 0755); err != nil {
		writeStatusError(w, http.StatusInternalServerError, "DIRECTORY_CREATION_FAILED", "Failed to create uploads directory")
		return
	}

	safeFilename := fmt.Sprintf("%s_%s", uuid.New().String(), header.Filename)
	fullPath := fmt.Sprintf("%s/%s", dir, safeFilename)

	out, err := os.Create(fullPath)
	if err != nil {
		writeStatusError(w, http.StatusInternalServerError, "FILE_SAVE_FAILED", "Failed to save file on server")
		return
	}
	defer out.Close()

	size, err := io.Copy(out, file)
	if err != nil {
		writeStatusError(w, http.StatusInternalServerError, "FILE_WRITE_FAILED", "Failed to write file contents")
		return
	}

	_, err = s.db.Exec(r.Context(), `
		INSERT INTO receipts (expense_id, file_path, file_size, mime_type)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (expense_id) 
		DO UPDATE SET file_path = EXCLUDED.file_path, file_size = EXCLUDED.file_size, mime_type = EXCLUDED.mime_type, uploaded_at = NOW()
	`, e.ID, fullPath, size, header.Header.Get("Content-Type"))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"receipt_url": "/assets/receipts/" + safeFilename,
		"file_size":   size,
	})
}
