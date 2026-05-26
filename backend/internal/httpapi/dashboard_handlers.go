package httpapi

import (
	"net/http"
	"strings"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/middleware"
)

func (s *Server) registerDashboardRoutes() {
	s.mux.Handle("/api/v1/dashboard/summary", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleDashboardSummary)))
	s.mux.Handle("/api/v1/dashboard/widgets", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleDashboardWidgets)))
}

func (s *Server) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	defaultCurrency := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("default_currency")))
	summary, err := s.reportService.DashboardSummaryForCurrency(r.Context(), userID, defaultCurrency)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleDashboardWidgets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	defaultCurrency := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("default_currency")))
	widgets, err := s.reportService.DashboardWidgetsForCurrency(r.Context(), userID, defaultCurrency)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, widgets)
}
