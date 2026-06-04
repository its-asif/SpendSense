package httpapi

import (
	"net/http"
	"strings"
	"time"

	"spendsense-backend/internal/category"
	"spendsense-backend/internal/middleware"

	"github.com/google/uuid"
)

type categoryRequest struct {
	Name  string  `json:"name"`
	Icon  *string `json:"icon,omitempty"`
	Color *string `json:"color,omitempty"`
	Kind  string  `json:"kind,omitempty"`
}

type categoryResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Icon      *string `json:"icon,omitempty"`
	Color     *string `json:"color,omitempty"`
	Kind      string  `json:"kind"`
	IsDefault bool    `json:"is_default"`
	IsOwned   bool    `json:"is_owned"`
	CreatedAt string  `json:"created_at"`
}

func (s *Server) registerCategoryRoutes() {
	s.mux.Handle("/api/v1/categories", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleCreateListCategories)))
	s.mux.Handle("/api/v1/categories/", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleCategoryByID)))
}

func (s *Server) handleCreateListCategories(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	switch r.Method {
	case http.MethodPost:
		var req categoryRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}
		created, err := s.categoryService.CreateCategory(r.Context(), userID, category.CreateRequest{
			Name: req.Name, Icon: req.Icon, Color: req.Color, Kind: req.Kind,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toCategoryResponse(created, userID))
		return
	case http.MethodGet:
		kind := strings.TrimSpace(r.URL.Query().Get("kind"))
		list, err := s.categoryService.ListCategories(r.Context(), userID, kind)
		if err != nil {
			writeError(w, err)
			return
		}
		resp := make([]categoryResponse, 0, len(list))
		for _, it := range list {
			resp = append(resp, toCategoryResponse(it, userID))
		}
		writeJSON(w, http.StatusOK, map[string]any{"categories": resp})
		return
	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
}

func (s *Server) handleCategoryByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/categories/")
	idStr = strings.Trim(idStr, "/")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_ID", "Invalid category id")
		return
	}
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))

	switch r.Method {
	case http.MethodGet:
		c, err := s.categoryService.GetCategory(r.Context(), userID, id, kind)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toCategoryResponse(c, userID))
		return
	case http.MethodPut:
		var req categoryRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}
		updated, err := s.categoryService.UpdateCategory(r.Context(), userID, id, category.UpdateRequest{
			Name: req.Name, Icon: req.Icon, Color: req.Color,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toCategoryResponse(updated, userID))
		return
	case http.MethodDelete:
		if err := s.categoryService.DeleteCategory(r.Context(), userID, id); err != nil {
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

func toCategoryResponse(c *category.Category, viewerID uuid.UUID) categoryResponse {
	owned := c.UserID != nil && *c.UserID == viewerID
	return categoryResponse{
		ID:        c.ID.String(),
		Name:      c.Name,
		Icon:      c.Icon,
		Color:     c.Color,
		Kind:      c.Kind,
		IsDefault: c.IsDefault,
		IsOwned:   owned,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
}
