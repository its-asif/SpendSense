package domain

import "fmt"

type ErrorCode string

const (
	// Auth errors
	ErrInvalidEmail   ErrorCode = "INVALID_EMAIL"
	ErrWeakPassword   ErrorCode = "WEAK_PASSWORD"
	ErrDuplicateEmail ErrorCode = "DUPLICATE_EMAIL"
	ErrUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrForbidden      ErrorCode = "FORBIDDEN"

	// Validation errors
	ErrInvalidAmount     ErrorCode = "INVALID_AMOUNT"
	ErrInvalidDate       ErrorCode = "INVALID_DATE"
	ErrInvalidCategory   ErrorCode = "INVALID_CATEGORY"
	ErrInvalidWallet     ErrorCode = "INVALID_WALLET"
	ErrInvalidCurrency   ErrorCode = "INVALID_CURRENCY"
	ErrInvalidPagination ErrorCode = "INVALID_PAGINATION"

	// Resource errors
	ErrNotFound      ErrorCode = "NOT_FOUND"
	ErrAlreadyExists ErrorCode = "ALREADY_EXISTS"

	// Server
	ErrInternal ErrorCode = "INTERNAL_ERROR"
)

type DomainError struct {
	Code       ErrorCode
	Message    string
	Field      *string
	StatusCode int
}

func (e *DomainError) Error() string {
	return e.Message
}

func NewDomainError(code ErrorCode, message string, statusCode int) *DomainError {
	return &DomainError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

func NewDomainErrorWithField(code ErrorCode, message string, field string, statusCode int) *DomainError {
	return &DomainError{
		Code:       code,
		Message:    message,
		Field:      &field,
		StatusCode: statusCode,
	}
}

func (e *DomainError) String() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
