package auth

import "testing"

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Fatal("expected error for short password")
	}
	if err := ValidatePassword("longenough"); err != nil {
		t.Fatalf("unexpected error for valid password: %v", err)
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	pass := "s3cur3P@ssw0rd"
	hash, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}
	if !VerifyPassword(hash, pass) {
		t.Fatal("password verification failed")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatal("expected verification to fail for wrong password")
	}
}
