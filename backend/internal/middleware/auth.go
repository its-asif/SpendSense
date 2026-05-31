package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"spendsense-backend/internal/auth"
	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

type contextKey string

const (
	userIDContextKey    contextKey = "userID"
	emailContextKey     contextKey = "email"
	sessionIDContextKey contextKey = "sessionID"
)

type AuthMiddleware struct {
	jwtManager       *auth.JWTManager
	sessionValidator SessionValidator
}

type SessionValidator interface {
	HasActiveSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (bool, error)
}

func NewAuthMiddleware(jwtManager *auth.JWTManager, sessionValidator SessionValidator) *AuthMiddleware {
	return &AuthMiddleware{jwtManager: jwtManager, sessionValidator: sessionValidator}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := m.authenticate(r)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		if claims.SessionID == uuid.Nil || m.sessionValidator == nil {
			writeAuthError(w, domain.NewDomainError(domain.ErrUnauthorized, "Session is no longer active", http.StatusUnauthorized))
			return
		}

		active, err := m.sessionValidator.HasActiveSession(r.Context(), claims.UserID, claims.SessionID)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		if !active {
			writeAuthError(w, domain.NewDomainError(domain.ErrUnauthorized, "Session is no longer active", http.StatusUnauthorized))
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
		ctx = context.WithValue(ctx, emailContextKey, claims.Email)
		ctx = context.WithValue(ctx, sessionIDContextKey, claims.SessionID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	return userID, ok
}

func EmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(emailContextKey).(string)
	return email, ok
}

func SessionIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	sessionID, ok := ctx.Value(sessionIDContextKey).(uuid.UUID)
	return sessionID, ok
}

func (m *AuthMiddleware) authenticate(r *http.Request) (*auth.Claims, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return nil, domain.NewDomainError(domain.ErrUnauthorized, "Authorization header is required", http.StatusUnauthorized)
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, domain.NewDomainError(domain.ErrUnauthorized, "Authorization header must use Bearer token", http.StatusUnauthorized)
	}

	tokenString := strings.TrimSpace(parts[1])
	if tokenString == "" {
		return nil, domain.NewDomainError(domain.ErrUnauthorized, "Bearer token is required", http.StatusUnauthorized)
	}

	claims, err := m.jwtManager.VerifyToken(tokenString)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func writeAuthError(w http.ResponseWriter, err error) {
	var domainErr *domain.DomainError
	if errors.As(err, &domainErr) {
		writeJSON(w, domainErr.StatusCode, map[string]any{
			"error": map[string]any{
				"code":    string(domainErr.Code),
				"message": domainErr.Message,
				"field":   domainErr.Field,
			},
		})
		return
	}

	writeJSON(w, http.StatusUnauthorized, map[string]any{
		"error": map[string]any{
			"code":    string(domain.ErrUnauthorized),
			"message": "Unauthorized",
		},
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
