package auth

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateAndVerifyToken(t *testing.T) {
	jm := NewJWTManager("test-secret")
	uid := uuid.New()
	email := "user@example.com"

	tok, err := jm.GenerateAccessToken(uid, email)
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
}
