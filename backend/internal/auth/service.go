package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/infra"

	"github.com/google/uuid"
)

type UserStore interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error)
	UpdateUserProfile(ctx context.Context, userID uuid.UUID, displayName, avatarURL string) (*domain.User, error)
	UpdateUserPreferences(ctx context.Context, userID uuid.UUID, baseCurrency, timezone, locale string) (*domain.User, error)
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, newHash string) error
	ListRefreshTokens(ctx context.Context, userID uuid.UUID) ([]infra.RefreshTokenRow, error)
	DeleteRefreshTokenByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
	DeleteOtherRefreshTokens(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error
	StoreRefreshToken(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, token string, expiresInHours int, device, ipAddress, userAgent string) error
	ValidateRefreshToken(ctx context.Context, userID uuid.UUID, token string) (uuid.UUID, bool, error)
	DeleteRefreshToken(ctx context.Context, userID uuid.UUID, token string) error
	DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredRefreshTokens(ctx context.Context) (int64, error)
	// TOTP 2FA
	SetTOTPSecret(ctx context.Context, userID uuid.UUID, secret string) error
	EnableTOTP(ctx context.Context, userID uuid.UUID) error
	DisableTOTP(ctx context.Context, userID uuid.UUID) error
	GetTOTPSecret(ctx context.Context, userID uuid.UUID) (string, bool, error)
}

type SessionMetadata struct {
	Device    string
	IPAddress string
	UserAgent string
}

type AuthService struct {
	db         UserStore
	jwtManager *JWTManager
}

func NewAuthService(db UserStore, jwtManager *JWTManager) *AuthService {
	return &AuthService{db: db, jwtManager: jwtManager}
}

func (as *AuthService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return as.db.GetUserByEmail(ctx, email)
}

func (as *AuthService) UpdateUserProfile(ctx context.Context, userID uuid.UUID, displayName, avatarURL string) (*domain.User, error) {
	displayName = strings.TrimSpace(displayName)
	avatarURL = strings.TrimSpace(avatarURL)

	updated, err := as.db.UpdateUserProfile(ctx, userID, displayName, avatarURL)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrInternal, "Failed to update user profile", 500)
	}

	return updated, nil
}

func (as *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := as.db.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return domain.NewDomainError(domain.ErrNotFound, "User not found", 404)
	}

	if !VerifyPassword(user.PasswordHash, oldPassword) {
		return domain.NewDomainError(domain.ErrUnauthorized, "Invalid current password", 401)
	}

	if de := ValidatePassword(newPassword); de != nil {
		return de
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return domain.NewDomainError(domain.ErrInternal, "Failed to hash password", 500)
	}

	if err := as.db.UpdateUserPassword(ctx, userID, hash); err != nil {
		return domain.NewDomainError(domain.ErrInternal, "Failed to update password", 500)
	}

	return nil
}

func (as *AuthService) UpdateUserPreferences(ctx context.Context, userID uuid.UUID, baseCurrency, timezone, locale string) (*domain.User, error) {
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	timezone = strings.TrimSpace(timezone)
	locale = strings.TrimSpace(locale)

	if baseCurrency == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidCurrency, "Base currency is required", 400)
	}
	if timezone == "" {
		timezone = "UTC"
	}
	if locale == "" {
		locale = "en-US"
	}

	updated, err := as.db.UpdateUserPreferences(ctx, userID, baseCurrency, timezone, locale)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrInternal, "Failed to update user preferences", 500)
	}

	return updated, nil
}

func (as *AuthService) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return as.db.GetUserByID(ctx, userID)
}

type RegisterRequest struct {
	Email    string
	Password string
}

type AuthResponse struct {
	AccessToken  string
	RefreshToken string
	User         *domain.User
}

func (as *AuthService) Register(ctx context.Context, req RegisterRequest, metadata SessionMetadata) (*AuthResponse, error) {
	fmt.Println("req:", req)
	if req.Email == "" {
		return nil, domain.NewDomainError(domain.ErrInvalidEmail, "Email is required", 400)
	}

	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Check if email already exists
	existing, _ := as.db.GetUserByEmail(ctx, req.Email)
	if existing != nil {
		return nil, domain.NewDomainError(domain.ErrDuplicateEmail, "Email already registered", 409)
	}

	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrInternal, "Failed to hash password", 500)
	}

	user := &domain.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hash,
		BaseCurrency: "USD",
		Timezone:     "UTC",
		Locale:       "en-US",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := as.db.CreateUser(ctx, user); err != nil {
		return nil, domain.NewDomainError(domain.ErrInternal, "Failed to create user", 500)
	}

	sessionID := uuid.New()
	accessToken, err := as.jwtManager.GenerateAccessToken(user.ID, user.Email, sessionID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrInternal, "Failed to generate token", 500)
	}

	refreshToken := as.jwtManager.GenerateRefreshToken()

	if err := as.db.StoreRefreshToken(ctx, user.ID, sessionID, refreshToken, 7*24, metadata.Device, metadata.IPAddress, metadata.UserAgent); err != nil {
		return nil, domain.NewDomainError(domain.ErrInternal, "Failed to store refresh token", 500)
	}

	return &AuthResponse{AccessToken: accessToken, RefreshToken: refreshToken, User: user}, nil
}

func (as *AuthService) Login(ctx context.Context, email, password string, metadata SessionMetadata) (*AuthResponse, error) {
	user, err := as.db.GetUserByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, domain.NewDomainError(domain.ErrUnauthorized, "Invalid email or password", 401)
	}

	if !VerifyPassword(user.PasswordHash, password) {
		return nil, domain.NewDomainError(domain.ErrUnauthorized, "Invalid email or password", 401)
	}

	sessionID := uuid.New()
	accessToken, err := as.jwtManager.GenerateAccessToken(user.ID, user.Email, sessionID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrInternal, "Failed to generate token", 500)
	}

	refreshToken := as.jwtManager.GenerateRefreshToken()
	if err := as.db.StoreRefreshToken(ctx, user.ID, sessionID, refreshToken, 7*24, metadata.Device, metadata.IPAddress, metadata.UserAgent); err != nil {
		return nil, domain.NewDomainError(domain.ErrInternal, "Failed to store refresh token", 500)
	}

	return &AuthResponse{AccessToken: accessToken, RefreshToken: refreshToken, User: user}, nil
}

func (as *AuthService) RefreshAccessToken(ctx context.Context, userID uuid.UUID, refreshToken string) (string, error) {
	sessionID, valid, err := as.db.ValidateRefreshToken(ctx, userID, refreshToken)
	if err != nil || !valid {
		return "", domain.NewDomainError(domain.ErrUnauthorized, "Invalid or expired refresh token", 401)
	}

	user, err := as.db.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return "", domain.NewDomainError(domain.ErrNotFound, "User not found", 404)
	}

	accessToken, err := as.jwtManager.GenerateAccessToken(user.ID, user.Email, sessionID)
	if err != nil {
		return "", domain.NewDomainError(domain.ErrInternal, "Failed to generate token", 500)
	}

	return accessToken, nil
}

// Logout removes a refresh token so it can no longer be used
func (as *AuthService) Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	if err := as.db.DeleteRefreshToken(ctx, userID, refreshToken); err != nil {
		return domain.NewDomainError(domain.ErrInternal, "Failed to delete refresh token", 500)
	}
	return nil
}

// LogoutAllSessions revokes all refresh tokens for a user.
func (as *AuthService) LogoutAllSessions(ctx context.Context, userID uuid.UUID) error {
	if err := as.db.DeleteAllRefreshTokens(ctx, userID); err != nil {
		return domain.NewDomainError(domain.ErrInternal, "Failed to delete user refresh tokens", 500)
	}
	return nil
}

// LogoutOtherSessions revokes all sessions except the current one.
func (as *AuthService) LogoutOtherSessions(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	if err := as.db.DeleteOtherRefreshTokens(ctx, userID, sessionID); err != nil {
		return domain.NewDomainError(domain.ErrInternal, "Failed to delete other user refresh tokens", 500)
	}
	return nil
}

// CleanupExpiredRefreshTokens removes expired refresh tokens and returns deleted rows.
func (as *AuthService) CleanupExpiredRefreshTokens(ctx context.Context) (int64, error) {
	deleted, err := as.db.DeleteExpiredRefreshTokens(ctx)
	if err != nil {
		return 0, domain.NewDomainError(domain.ErrInternal, "Failed to cleanup expired refresh tokens", 500)
	}
	return deleted, nil
}
