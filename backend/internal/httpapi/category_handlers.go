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
}

type categoryResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Icon      *string `json:"icon,omitempty"`
	Color     *string `json:"color,omitempty"`
	IsDefault bool    `json:"is_default"`
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
		created, err := s.categoryService.CreateCategory(r.Context(), userID, category.CreateRequest{Name: req.Name, Icon: req.Icon, Color: req.Color})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, toCategoryResponse(created))
		return
	case http.MethodGet:
		list, err := s.categoryService.ListCategories(r.Context(), userID)
		if err != nil {
			writeError(w, err)
			return
		}
		resp := make([]categoryResponse, 0, len(list))
		for _, it := range list {
			resp = append(resp, toCategoryResponse(it))
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

	switch r.Method {
	case http.MethodGet:
		c, err := s.categoryService.GetCategory(r.Context(), id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toCategoryResponse(c))
		return
	case http.MethodPut:
		var req categoryRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeRequestError(w, err)
			return
		}
		updated, err := s.categoryService.UpdateCategory(r.Context(), userID, id, category.UpdateRequest{Name: req.Name, Icon: req.Icon, Color: req.Color})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, toCategoryResponse(updated))
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

func toCategoryResponse(c *category.Category) categoryResponse {
	var uid string
	if c.ID != uuid.Nil {
		uid = c.ID.String()
	}
	return categoryResponse{ID: uid, Name: c.Name, Icon: c.Icon, Color: c.Color, IsDefault: c.IsDefault, CreatedAt: c.CreatedAt.Format(time.RFC3339)}
}
