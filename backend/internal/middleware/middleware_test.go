package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"spendsense-backend/internal/auth"

	"github.com/google/uuid"
)

func TestCORS(t *testing.T) {
	origins := []string{"https://example.com", "*"}
	handler := CORS(origins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 1. Check Preflight (OPTIONS)
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 No Content for OPTIONS, got %d", rec.Code)
	}

	// 2. Check Allowed Origin headers
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" { // "*" takes priority because allowAll is true
		t.Errorf("expected * allowed origin, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSSpecificOrigin(t *testing.T) {
	origins := []string{"https://specific.com"}
	handler := CORS(origins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://specific.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://specific.com" {
		t.Errorf("expected specific origin header, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestRecoverer(t *testing.T) {
	handler := Recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong!")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Ensure the panic is caught and returns 500
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "INTERNAL_ERROR" {
		t.Errorf("unexpected error code: %s", errObj["code"])
	}
}

func TestRequestLogger(t *testing.T) {
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 Created, got %d", rec.Code)
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, time.Second)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Call 1: Success (count = 1)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Call 2: Success (count = 2)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Call 3: Rate Limited (count = 3, limit = 2)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode rate limit body: %v", err)
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "RATE_LIMITED" {
		t.Errorf("expected RATE_LIMITED error code, got %s", errObj["code"])
	}
}

type fakeSessionValidator struct {
	active bool
	err    error
}

func (f *fakeSessionValidator) HasActiveSession(ctx context.Context, userID, sessionID uuid.UUID) (bool, error) {
	return f.active, f.err
}

func TestAuthMiddleware(t *testing.T) {
	jwtMgr := auth.NewJWTManager("test-secret-key-12345678901234567890")
	validator := &fakeSessionValidator{active: true}
	mw := NewAuthMiddleware(jwtMgr, validator)

	nextCalled := false
	var ctxUserID uuid.UUID
	var ctxEmail string
	var ctxSessionID uuid.UUID

	handler := mw.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		ctxUserID, _ = UserIDFromContext(r.Context())
		ctxEmail, _ = EmailFromContext(r.Context())
		ctxSessionID, _ = SessionIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// 1. Missing Authorization header -> 401
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing auth header, got %d", rec.Code)
	}

	// 2. Invalid Format -> 401
	req = httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Token something")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid header prefix, got %d", rec.Code)
	}

	// 3. Valid Token, Active Session -> Success
	userID := uuid.New()
	sessionID := uuid.New()
	token, err := jwtMgr.GenerateAccessToken(userID, "user@example.com", sessionID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for valid token, got %d", rec.Code)
	}
	if !nextCalled {
		t.Errorf("expected next handler to be called")
	}
	if ctxUserID != userID || ctxEmail != "user@example.com" || ctxSessionID != sessionID {
		t.Errorf("context values did not match generated token claims")
	}

	// 4. Valid Token, Inactive Session -> 401
	validator.active = false
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for inactive session, got %d", rec.Code)
	}
}
