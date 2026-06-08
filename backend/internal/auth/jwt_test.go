package auth

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateAndVerifyToken(t *testing.T) {
	jm := NewJWTManager("test-secret-key-12345678901234567890")
	uid := uuid.New()
	email := "user@example.com"
	sessionID := uuid.New()

	tok, err := jm.GenerateAccessToken(uid, email, sessionID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := jm.VerifyToken(tok)
	if err != nil {
		t.Fatalf("failed to verify token: %v", err)
	}

	if claims.UserID != uid {
		t.Fatalf("expected user id %s, got %s", uid, claims.UserID)
	}
	if claims.Email != email {
		t.Fatalf("expected email %s, got %s", email, claims.Email)
	}
	if claims.SessionID != sessionID {
		t.Fatalf("expected session id %s, got %s", sessionID, claims.SessionID)
	}

	// Test invalid token
	_, err = jm.VerifyToken("invalid-token-string")
	if err == nil {
		t.Errorf("expected error for invalid token string")
	}

	// Test token signed with different secret
	otherJm := NewJWTManager("different-secret-key-12345678901234567890")
	otherTok, err := otherJm.GenerateAccessToken(uid, email, sessionID)
	if err != nil {
		t.Fatalf("failed to generate other token: %v", err)
	}

	_, err = jm.VerifyToken(otherTok)
	if err == nil {
		t.Errorf("expected verification error for token signed with a different key")
	}

	// Test empty refresh token generator returns valid string
	ref := jm.GenerateRefreshToken()
	if ref == "" {
		t.Errorf("expected refresh token to be a non-empty string")
	}
}
