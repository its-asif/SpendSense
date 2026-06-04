package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"spendsense-backend/internal/middleware"
	"spendsense-backend/internal/notification"

	"github.com/google/uuid"
)

type notificationResponse struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	IsRead    bool            `json:"is_read"`
	CreatedAt string          `json:"created_at"`
}

func (s *Server) registerNotificationRoutes() {
	s.mux.Handle("/api/v1/notifications", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleNotifications)))
	s.mux.Handle("/api/v1/notifications/read-all", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleNotificationsReadAll)))
	s.mux.Handle("/api/v1/notifications/", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleNotificationByID)))
}

func (s *Server) handleNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		limit := 30
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				limit = parsed
			}
		}
		unreadOnly := strings.EqualFold(r.URL.Query().Get("unread_only"), "true")

		_ = s.notificationService.RunChecks(r.Context(), userID)

		items, err := s.notificationService.List(r.Context(), userID, limit, unreadOnly)
		if err != nil {
			writeError(w, err)
			return
		}
		unreadCount, err := s.notificationService.CountUnread(r.Context(), userID)
		if err != nil {
			writeError(w, err)
			return
		}

		resp := make([]notificationResponse, 0, len(items))
		for _, item := range items {
			resp = append(resp, toNotificationResponse(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"notifications": resp,
			"unread_count":  unreadCount,
		})
		return
	default:
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}
}

func (s *Server) handleNotificationsReadAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
	if err := s.notificationService.MarkAllRead(r.Context(), userID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleNotificationByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/notifications/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeStatusError(w, http.StatusNotFound, "NOT_FOUND", "Not found")
		return
	}

	id, err := uuid.Parse(parts[0])
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_ID", "Invalid notification id")
		return
	}

	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	switch action {
	case "read":
		if err := s.notificationService.MarkRead(r.Context(), userID, id); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case "dismiss":
		if err := s.notificationService.Dismiss(r.Context(), userID, id); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		writeStatusError(w, http.StatusNotFound, "NOT_FOUND", "Not found")
	}
}

func (s *Server) scheduleNotificationChecks(userID uuid.UUID) {
	if s.notificationService == nil {
		return
	}
	go func() {
		_ = s.notificationService.RunChecks(context.Background(), userID)
	}()
}

func toNotificationResponse(n *notification.Notification) notificationResponse {
	return notificationResponse{
		ID:        n.ID.String(),
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		Metadata:  n.Metadata,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}
}
