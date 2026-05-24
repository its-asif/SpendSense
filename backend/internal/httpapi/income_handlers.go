package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/income"
	"spendsense-backend/internal/middleware"

	"github.com/google/uuid"
)

type createIncomeRequest struct {
	WalletID   string  `json:"wallet_id"`
	CategoryID *string `json:"category_id,omitempty"`
	SourceName string  `json:"source_name"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	IncomeDate string  `json:"income_date"`
	Notes      *string `json:"notes,omitempty"`
}

type incomeResponse struct {
	ID         uuid.UUID  `json:"id"`
	WalletID   uuid.UUID  `json:"wallet_id"`
	CategoryID *uuid.UUID `json:"category_id,omitempty"`
	SourceName string     `json:"source_name"`
	Amount     float64    `json:"amount"`
	Currency   string     `json:"currency"`
	IncomeDate string     `json:"income_date"`
	Notes      *string    `json:"notes,omitempty"`
	CreatedAt  string     `json:"created_at"`
	UpdatedAt  string     `json:"updated_at"`
}

func (s *Server) registerIncomeRoutes() {
	s.mux.Handle("/api/v1/incomes", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleCreateListIncomes)))
	s.mux.Handle("/api/v1/incomes/", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleIncomeByID)))
}

func (s *Server) handleCreateListIncomes(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req createIncomeRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}

		walletUUID, _ := uuid.Parse(req.WalletID)
		var categoryUUID *uuid.UUID
		if req.CategoryID != nil && strings.TrimSpace(*req.CategoryID) != "" {
			parsed, _ := uuid.Parse(*req.CategoryID)
			categoryUUID = &parsed
		}
		parsedDate, _ := time.Parse("2006-01-02", req.IncomeDate)

		svcReq := income.CreateRequest{
			WalletID:   walletUUID,
			CategoryID: categoryUUID,
			SourceName: req.SourceName,
			Amount:     req.Amount,
			Currency:   req.Currency,
			IncomeDate: parsedDate,
			Notes:      req.Notes,
		}

		created, err := s.incomeService.CreateIncome(r.Context(), userID, svcReq)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, toIncomeResponse(created))
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

		list, next, err := s.incomeService.ListIncomes(r.Context(), userID, limit, cursor)
		if err != nil {
			writeError(w, err)
			return
		}

		resp := make([]incomeResponse, 0, len(list))
		for _, it := range list {
			resp = append(resp, toIncomeResponse(it))
		}

		writeJSON(w, http.StatusOK, map[string]any{"incomes": resp, "next_cursor": next})
		return
	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
}

func (s *Server) handleIncomeByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/incomes/")
	idStr = strings.Trim(idStr, "/")
	incomeID, err := uuid.Parse(idStr)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_INCOME_ID", "Invalid income id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, err := s.incomeService.GetIncome(r.Context(), userID, incomeID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toIncomeResponse(item))
		return
	case http.MethodPut:
		var req createIncomeRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}

		walletUUID, _ := uuid.Parse(req.WalletID)
		var categoryUUID *uuid.UUID
		if req.CategoryID != nil && strings.TrimSpace(*req.CategoryID) != "" {
			parsed, _ := uuid.Parse(*req.CategoryID)
			categoryUUID = &parsed
		}
		parsedDate, _ := time.Parse("2006-01-02", req.IncomeDate)

		svcReq := income.UpdateRequest{
			WalletID:   walletUUID,
			CategoryID: categoryUUID,
			SourceName: req.SourceName,
			Amount:     req.Amount,
			Currency:   req.Currency,
			IncomeDate: parsedDate,
			Notes:      req.Notes,
		}

		updated, err := s.incomeService.UpdateIncome(r.Context(), userID, incomeID, svcReq)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toIncomeResponse(updated))
		return
	case http.MethodDelete:
		if err := s.incomeService.SoftDeleteIncome(r.Context(), userID, incomeID); err != nil {
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

func toIncomeResponse(item *income.Income) incomeResponse {
	var dateStr string
	if !item.IncomeDate.IsZero() {
		dateStr = item.IncomeDate.Format("2006-01-02")
	}
	return incomeResponse{
		ID:         item.ID,
		WalletID:   item.WalletID,
		CategoryID: item.CategoryID,
		SourceName: item.SourceName,
		Amount:     item.Amount,
		Currency:   item.Currency,
		IncomeDate: dateStr,
		Notes:      item.Notes,
		CreatedAt:  item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  item.UpdatedAt.Format(time.RFC3339),
	}
}
