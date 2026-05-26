package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"spendsense-backend/internal/auth"
	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/middleware"

	"github.com/google/uuid"
)

type errorResponse struct {
	Error responseErrorBody `json:"error"`
}

type responseErrorBody struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Field   *string `json:"field,omitempty"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	Email        string `json:"email"`
	RefreshToken string `json:"refresh_token"`
}

type userResponse struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	DisplayName  *string   `json:"display_name,omitempty"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	BaseCurrency string    `json:"base_currency"`
	Timezone     string    `json:"timezone"`
	Locale       string    `json:"locale"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
}

type authResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	User         userResponse `json:"user,omitempty"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
}

type currentUserResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type updateUserPreferencesRequest struct {
	BaseCurrency string `json:"base_currency"`
	Timezone     string `json:"timezone"`
	Locale       string `json:"locale"`
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", handleHealth)
	s.mux.HandleFunc("/openapi.yaml", s.handleOpenAPISpec)
	s.mux.HandleFunc("/api/docs", s.handleSwaggerUI)
	s.mux.HandleFunc("/api/v1/currencies", s.handleCurrencyList)
	s.mux.HandleFunc("/api/v1/currencies/convert", s.handleCurrencyConvert)
	s.registerDashboardRoutes()
	s.mux.HandleFunc("/auth/register", s.handleRegister)
	s.mux.HandleFunc("/auth/login", s.handleLogin)
	s.mux.HandleFunc("/auth/refresh", s.handleRefresh)
	s.mux.Handle("/auth/logout", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleLogout)))
	s.mux.Handle("/auth/logout-all", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleLogoutAll)))
	s.mux.Handle("/auth/me", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleMe)))
	s.mux.Handle("/auth/me/preferences", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleUpdateMePreferences)))
	s.mux.Handle("/ops/refresh-tokens/cleanup", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleCleanupRefreshTokens)))
	s.registerExpenseRoutes()
	s.registerIncomeRoutes()
	s.registerWalletRoutes()
	s.registerCategoryRoutes()
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	var req registerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	resp, err := s.authService.Register(r.Context(), auth.RegisterRequest{
		Email:    strings.TrimSpace(req.Email),
		Password: req.Password,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, authResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User:         toUserResponse(resp.User),
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	resp, err := s.authService.Login(r.Context(), strings.TrimSpace(req.Email), req.Password)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User:         toUserResponse(resp.User),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	var req refreshRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	user, err := s.authService.GetUserByEmail(r.Context(), strings.TrimSpace(req.Email))
	if err != nil || user == nil {
		writeStatusError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	accessToken, err := s.authService.RefreshAccessToken(r.Context(), user.ID, strings.TrimSpace(req.RefreshToken))
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, tokenResponse{AccessToken: accessToken})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusInternalServerError, string(domain.ErrInternal), "Missing authenticated user")
		return
	}

	email, _ := middleware.EmailFromContext(r.Context())
	writeJSON(w, http.StatusOK, currentUserResponse{
		UserID: userID.String(),
		Email:  email,
	})
}

func (s *Server) handleUpdateMePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	var req updateUserPreferencesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	updated, err := s.authService.UpdateUserPreferences(r.Context(), userID, req.BaseCurrency, req.Timezone, req.Locale)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(updated))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	if req.RefreshToken == "" {
		writeStatusError(w, http.StatusBadRequest, "INVALID_REQUEST", "refresh_token is required")
		return
	}

	if err := s.authService.Logout(r.Context(), userID, req.RefreshToken); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	if err := s.authService.LogoutAllSessions(r.Context(), userID); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleCleanupRefreshTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	deleted, err := s.authService.CleanupExpiredRefreshTokens(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}

/*
Decodes JSON from the request body into the provided destination struct.

- dst should be a pointer to a struct where the decoded data will be stored.
- The request body is limited to 1MB to prevent abuse.
- Unknown fields in the JSON will cause an error, ensuring that only expected data is processed.
*/
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	defer r.Body.Close()
	// Limit request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return fmt.Errorf("unexpected extra JSON content")
	} else if !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}

func writeRequestError(w http.ResponseWriter, err error) {
	message := "Invalid request body"
	if err != nil {
		message = err.Error()
	}
	writeStatusError(w, http.StatusBadRequest, "INVALID_REQUEST", message)
}

func writeStatusError(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, errorResponse{
		Error: responseErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

func writeError(w http.ResponseWriter, err error) {
	var domainErr *domain.DomainError
	if errors.As(err, &domainErr) {
		statusCode := domainErr.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}

		writeJSON(w, statusCode, errorResponse{
			Error: responseErrorBody{
				Code:    string(domainErr.Code),
				Message: domainErr.Message,
				Field:   domainErr.Field,
			},
		})
		return
	}

	writeStatusError(w, http.StatusInternalServerError, string(domain.ErrInternal), "Internal server error")
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

func toUserResponse(user *domain.User) userResponse {
	if user == nil {
		return userResponse{}
	}

	return userResponse{
		ID:           user.ID,
		Email:        user.Email,
		DisplayName:  user.DisplayName,
		AvatarURL:    user.AvatarURL,
		BaseCurrency: user.BaseCurrency,
		Timezone:     user.Timezone,
		Locale:       user.Locale,
		CreatedAt:    user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    user.UpdatedAt.Format(time.RFC3339),
	}
}
