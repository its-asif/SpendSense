package httpapi

import (
	"net/http"
	"strings"
	"time"

	"spendsense-backend/internal/budget"
	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/middleware"

	"github.com/google/uuid"
)

const budgetDateFormat = "2006-01-02"

type budgetRequest struct {
	CategoryID      string  `json:"category_id"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	Period          string  `json:"period,omitempty"`
	StartDate       string  `json:"start_date,omitempty"`
	RolloverEnabled bool    `json:"rollover_enabled,omitempty"`
}

type budgetResponse struct {
	ID              string  `json:"id"`
	CategoryID      string  `json:"category_id"`
	CategoryName    string  `json:"category_name"`
	CategoryIcon    *string `json:"category_icon,omitempty"`
	CategoryColor   *string `json:"category_color,omitempty"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	Period          string  `json:"period"`
	StartDate       string  `json:"start_date"`
	RolloverEnabled bool    `json:"rollover_enabled"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func (s *Server) registerBudgetRoutes() {
	s.mux.Handle("/api/v1/budgets", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleCreateListBudgets)))
	s.mux.Handle("/api/v1/budgets/", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleBudgetByID)))
}

func (s *Server) handleCreateListBudgets(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req budgetRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}
		svcReq, err := toBudgetCreateRequest(req)
		if err != nil {
			writeError(w, err)
			return
		}
		created, err := s.budgetService.Create(r.Context(), userID, svcReq)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toBudgetResponse(created))
		return
	case http.MethodGet:
		period := strings.TrimSpace(r.URL.Query().Get("period"))
		list, err := s.budgetService.List(r.Context(), userID, period)
		if err != nil {
			writeError(w, err)
			return
		}
		resp := make([]budgetResponse, 0, len(list))
		for _, item := range list {
			resp = append(resp, toBudgetResponse(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"budgets": resp})
		return
	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
}

func (s *Server) handleBudgetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/budgets/")
	idStr = strings.Trim(idStr, "/")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_ID", "Invalid budget id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, err := s.budgetService.Get(r.Context(), userID, id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toBudgetResponse(item))
		return
	case http.MethodPut:
		var req budgetRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}
		svcReq, err := toBudgetCreateRequest(req)
		if err != nil {
			writeError(w, err)
			return
		}
		updated, err := s.budgetService.Update(r.Context(), userID, id, budget.UpdateRequest{
			CategoryID:      svcReq.CategoryID,
			Amount:          svcReq.Amount,
			Currency:        svcReq.Currency,
			Period:          svcReq.Period,
			StartDate:       svcReq.StartDate,
			RolloverEnabled: svcReq.RolloverEnabled,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toBudgetResponse(updated))
		return
	case http.MethodDelete:
		if err := s.budgetService.Delete(r.Context(), userID, id); err != nil {
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

func toBudgetCreateRequest(req budgetRequest) (budget.CreateRequest, error) {
	categoryID, err := uuid.Parse(strings.TrimSpace(req.CategoryID))
	if err != nil {
		return budget.CreateRequest{}, domain.NewDomainErrorWithField(domain.ErrInvalidCategory, "invalid category id", "category_id", 400)
	}

	var startDate time.Time
	if strings.TrimSpace(req.StartDate) != "" {
		startDate, err = time.Parse(budgetDateFormat, req.StartDate)
		if err != nil {
			return budget.CreateRequest{}, domain.NewDomainErrorWithField(domain.ErrInvalidDate, "start_date must be YYYY-MM-DD", "start_date", 400)
		}
	}

	return budget.CreateRequest{
		CategoryID:      categoryID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Period:          req.Period,
		StartDate:       startDate,
		RolloverEnabled: req.RolloverEnabled,
	}, nil
}

func toBudgetResponse(b *budget.Budget) budgetResponse {
	return budgetResponse{
		ID:              b.ID.String(),
		CategoryID:      b.CategoryID.String(),
		CategoryName:    b.CategoryName,
		CategoryIcon:    b.CategoryIcon,
		CategoryColor:   b.CategoryColor,
		Amount:          b.Amount,
		Currency:        b.Currency,
		Period:          b.Period,
		StartDate:       b.StartDate.Format(budgetDateFormat),
		RolloverEnabled: b.RolloverEnabled,
		CreatedAt:       b.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       b.UpdatedAt.Format(time.RFC3339),
	}
}
