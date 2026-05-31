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

	"encoding/base64"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
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
	TOTPCode string `json:"totp_code,omitempty"`
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
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	SessionID   string `json:"session_id"`
	TOTPEnabled bool   `json:"totp_enabled"`
}

type updateUserPreferencesRequest struct {
	BaseCurrency string `json:"base_currency"`
	Timezone     string `json:"timezone"`
	Locale       string `json:"locale"`
}

type updateUserProfileRequest struct {
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type sessionMetadata struct {
	Device string
}

func detectSessionMetadata(r *http.Request) sessionMetadata {
	userAgent := r.UserAgent()
	lower := strings.ToLower(userAgent)
	// Allow clients (CLI) to explicitly provide OS info via headers
	osHint := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Client-OS")))
	if osHint == "" {
		// common alternates
		osHint = strings.ToLower(strings.TrimSpace(r.Header.Get("X-Platform")))
	}

	// osHint
	fmt.Println("userAgent:", userAgent)
	fmt.Println("osHint:", osHint)
	fmt.Println("lower:", lower)

	osName := "Unknown OS"
	if osHint != "" {
		switch {
		case strings.Contains(osHint, "iphone") || strings.Contains(osHint, "ipad") || strings.Contains(osHint, "ios"):
			osName = "iOS"
		case strings.Contains(osHint, "android"):
			osName = "Android"
		case strings.Contains(osHint, "mac") || strings.Contains(osHint, "darwin"):
			osName = "macOS"
		case strings.Contains(osHint, "win") || strings.Contains(osHint, "windows"):
			osName = "Windows"
		case strings.Contains(osHint, "linux"):
			osName = "Linux"
		default:
			osName = strings.Title(osHint)
		}
	} else {
		if strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad") {
			osName = "iOS"
		} else if strings.Contains(lower, "android") {
			osName = "Android"
		} else if strings.Contains(lower, "mac os") || strings.Contains(lower, "macintosh") {
			osName = "macOS"
		} else if strings.Contains(lower, "windows") {
			osName = "Windows"
		} else if strings.Contains(lower, "linux") {
			osName = "Linux"
		}
	}

	// Detect common CLI tools and HTTP libraries first
	switch {
	case strings.Contains(lower, "spendsense-cli") || strings.Contains(lower, "expense-tracker-cli"):
		return sessionMetadata{Device: "CLI (spendsense-cli) on " + osName}
	case strings.Contains(lower, "curl"):
		return sessionMetadata{Device: "CLI (curl) on " + osName}
	case strings.Contains(lower, "httpie"):
		return sessionMetadata{Device: "CLI (HTTPie) on " + osName}
	case strings.Contains(lower, "wget"):
		return sessionMetadata{Device: "CLI (wget) on " + osName}
	case strings.Contains(lower, "go-http-client") || strings.Contains(lower, "golang"):
		// If the client provided an explicit OS hint header use it, otherwise OS may be unknown
		return sessionMetadata{Device: "CLI (Go http client) on " + osName}
	case strings.Contains(lower, "python-requests"):
		return sessionMetadata{Device: "CLI (python-requests) on " + osName}
	case strings.Contains(lower, "okhttp"):
		return sessionMetadata{Device: "CLI (OkHttp) on " + osName}
	case strings.Contains(lower, "node-fetch") || strings.Contains(lower, "axios") || strings.Contains(lower, "nodejs"):
		return sessionMetadata{Device: "CLI (Node.js) on " + osName}
	}

	// Fallback to browser detection
	browser := "Unknown Browser"
	switch {
	case strings.Contains(lower, "edg/"):
		browser = "Edge"
	case strings.Contains(lower, "chrome/") && !strings.Contains(lower, "edg/"):
		browser = "Chrome"
	case strings.Contains(lower, "firefox/"):
		browser = "Firefox"
	case strings.Contains(lower, "safari/") && !strings.Contains(lower, "chrome/"):
		browser = "Safari"
	}

	return sessionMetadata{Device: browser + " on " + osName}
}

func detectClientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	remote := strings.TrimSpace(r.RemoteAddr)
	if idx := strings.LastIndex(remote, ":"); idx > 0 {
		return remote[:idx]
	}
	return remote
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
	s.mux.Handle("/auth/logout-other", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleLogoutOtherSessions)))
	s.mux.Handle("/auth/me", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleMe)))
	s.mux.Handle("/auth/me/profile", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleUpdateMeProfile)))
	s.mux.Handle("/auth/me/preferences", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleUpdateMePreferences)))
	s.mux.Handle("/auth/me/password", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleChangePassword)))
	s.mux.Handle("/auth/me/sessions", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleListSessions)))
	s.mux.Handle("/auth/me/sessions/", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handleDeleteSession)))
	s.mux.Handle("/auth/me/2fa/setup", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handle2FASetup)))
	s.mux.Handle("/auth/me/2fa/confirm", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handle2FAConfirm)))
	s.mux.Handle("/auth/me/2fa/disable", s.authMiddleware.RequireAuth(http.HandlerFunc(s.handle2FADisable)))
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

	metadata := detectSessionMetadata(r)
	resp, err := s.authService.Register(r.Context(), auth.RegisterRequest{
		Email:    strings.TrimSpace(req.Email),
		Password: req.Password,
	}, auth.SessionMetadata{Device: metadata.Device, IPAddress: detectClientIP(r), UserAgent: r.UserAgent()})
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

	// Pre-check user and password so we can enforce TOTP when enabled
	user, err := s.authService.GetUserByEmail(r.Context(), strings.TrimSpace(req.Email))
	if err != nil || user == nil {
		writeStatusError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	if !auth.VerifyPassword(user.PasswordHash, req.Password) {
		writeStatusError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// If the user has TOTP enabled, require a valid TOTP code
	if user.TOTPEnabled {
		if strings.TrimSpace(req.TOTPCode) == "" {
			writeStatusError(w, http.StatusUnauthorized, "TOTP_REQUIRED", "Two-factor authentication code required")
			return
		}

		secret, _, err := s.db.GetTOTPSecret(r.Context(), user.ID)
		if err != nil {
			writeError(w, err)
			return
		}
		if secret == "" {
			writeStatusError(w, http.StatusBadRequest, "TOTP_NOT_SETUP", "TOTP not setup for this account")
			return
		}

		valid, err := totp.ValidateCustom(strings.TrimSpace(req.TOTPCode), secret, time.Now(), totp.ValidateOpts{
			Period:    30,
			Skew:      1,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		if !valid {
			writeStatusError(w, http.StatusBadRequest, "INVALID_CODE", "Invalid TOTP code")
			return
		}
	}

	// Password (and TOTP, if required) validated: generate tokens via auth service
	metadata := detectSessionMetadata(r)
	resp, err := s.authService.Login(r.Context(), strings.TrimSpace(req.Email), req.Password, auth.SessionMetadata{Device: metadata.Device, IPAddress: detectClientIP(r), UserAgent: r.UserAgent()})
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
	sessionID, _ := middleware.SessionIDFromContext(r.Context())

	// fetch user to expose totp status
	user, err := s.authService.GetUserByID(r.Context(), userID)
	if err != nil || user == nil {
		writeStatusError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	writeJSON(w, http.StatusOK, currentUserResponse{
		UserID:      userID.String(),
		Email:       email,
		SessionID:   sessionID.String(),
		TOTPEnabled: user.TOTPEnabled,
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

func (s *Server) handleUpdateMeProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	var req updateUserProfileRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	updated, err := s.authService.UpdateUserProfile(r.Context(), userID, req.DisplayName, req.AvatarURL)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUserResponse(updated))
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	var req changePasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		writeStatusError(w, http.StatusBadRequest, "INVALID_REQUEST", "Both old_password and new_password are required")
		return
	}

	if err := s.authService.ChangePassword(r.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	if _, err := s.authService.CleanupExpiredRefreshTokens(r.Context()); err != nil {
		writeError(w, err)
		return
	}

	sessions, err := s.db.ListRefreshTokens(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	// path is /auth/me/sessions/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		writeStatusError(w, http.StatusBadRequest, "INVALID_REQUEST", "missing session id")
		return
	}
	idStr := parts[len(parts)-1]
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeStatusError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid session id")
		return
	}

	if err := s.db.DeleteRefreshTokenByID(r.Context(), userID, id); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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

func (s *Server) handleLogoutOtherSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	sessionID, ok := middleware.SessionIDFromContext(r.Context())
	if !ok || sessionID == uuid.Nil {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	if err := s.authService.LogoutOtherSessions(r.Context(), userID, sessionID); err != nil {
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

type twoFASetupResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otp_auth_url"`
	QRDataURL  string `json:"qr_data_url"`
}

type twoFAConfirmRequest struct {
	Code   string `json:"code"`
	Secret string `json:"secret,omitempty"`
}

// handle2FASetup generates a TOTP secret and returns a provisioning URI + QR code data URL
func (s *Server) handle2FASetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	email, _ := middleware.EmailFromContext(r.Context())

	key, err := totp.Generate(totp.GenerateOpts{Issuer: "SpendSense", AccountName: email})
	if err != nil {
		writeError(w, err)
		return
	}

	// store secret (base32)
	if err := s.db.SetTOTPSecret(r.Context(), userID, key.Secret()); err != nil {
		writeError(w, err)
		return
	}

	// generate QR PNG
	png, err := qrcode.Encode(key.URL(), qrcode.Medium, 256)
	if err != nil {
		writeError(w, err)
		return
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

	writeJSON(w, http.StatusOK, twoFASetupResponse{Secret: key.Secret(), OTPAuthURL: key.URL(), QRDataURL: dataURL})
}

// handle2FAConfirm verifies the provided TOTP code and enables 2FA
func (s *Server) handle2FAConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	var req twoFAConfirmRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeRequestError(w, err)
		return
	}

	secret, _, err := s.db.GetTOTPSecret(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	if secret == "" {
		secret = strings.TrimSpace(req.Secret)
	}
	if secret == "" {
		writeStatusError(w, http.StatusBadRequest, "TOTP_NOT_SETUP", "TOTP not setup for this account")
		return
	}

	if err := s.db.SetTOTPSecret(r.Context(), userID, secret); err != nil {
		writeError(w, err)
		return
	}

	expectedCode, err := totp.GenerateCodeCustom(secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      0,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	log.Printf("2FA confirm debug: userID=%s entered_code=%q expected_code=%q", userID.String(), strings.TrimSpace(req.Code), expectedCode)

	valid, err := totp.ValidateCustom(strings.TrimSpace(req.Code), secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if !valid {
		writeStatusError(w, http.StatusBadRequest, "INVALID_CODE", "Invalid TOTP code")
		return
	}

	if err := s.db.EnableTOTP(r.Context(), userID); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handle2FADisable disables TOTP for the authenticated user
func (s *Server) handle2FADisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeStatusError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeStatusError(w, http.StatusUnauthorized, string(domain.ErrUnauthorized), "Unauthorized")
		return
	}

	if err := s.db.DisableTOTP(r.Context(), userID); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
