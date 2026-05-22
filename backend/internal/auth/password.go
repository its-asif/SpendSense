package auth

import (
	"spendsense-backend/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hash), err
}

func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// checks password strength and returns a domain error when invalid
func ValidatePassword(password string) *domain.DomainError {
	if len(password) < 8 {
		return domain.NewDomainError(domain.ErrWeakPassword, "Password must be at least 8 characters", 400)
	}
	return nil
}
