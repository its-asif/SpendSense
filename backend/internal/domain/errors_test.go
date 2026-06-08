package domain

import (
	"testing"
)

func TestDomainError(t *testing.T) {
	// 1. Without field
	err := NewDomainError(ErrInvalidEmail, "invalid email format", 400)
	if err.Error() != "invalid email format" {
		t.Errorf("expected 'invalid email format', got '%s'", err.Error())
	}
	if err.String() != "INVALID_EMAIL: invalid email format" {
		t.Errorf("expected 'INVALID_EMAIL: invalid email format', got '%s'", err.String())
	}
	if err.Field != nil {
		t.Errorf("expected Field to be nil")
	}

	// 2. With field
	errWithField := NewDomainErrorWithField(ErrInvalidAmount, "amount is required", "amount", 400)
	if errWithField.Field == nil || *errWithField.Field != "amount" {
		t.Errorf("expected Field to be 'amount'")
	}
	if errWithField.String() != "INVALID_AMOUNT: amount is required" {
		t.Errorf("expected string representation, got '%s'", errWithField.String())
	}
}
