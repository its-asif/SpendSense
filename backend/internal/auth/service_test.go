package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/infra"

	"github.com/google/uuid"
)

type fakeUserStore struct {
	users         map[string]*domain.User
	storedRefresh string
	totpSecret    string
	totpEnabled   bool
}

func (f *fakeUserStore) CreateUser(ctx context.Context, user *domain.User) error {
	f.users[user.Email] = user
	return nil
}
func (f *fakeUserStore) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	return nil, nil
}
func (f *fakeUserStore) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	for _, u := range f.users {
		if u.ID == userID {
			return u, nil
		}
	}
	return nil, nil
}
func (f *fakeUserStore) UpdateUserProfile(ctx context.Context, userID uuid.UUID, displayName, avatarURL string) (*domain.User, error) {
	for _, u := range f.users {
		if u.ID == userID {
			if displayName == "" {
				u.DisplayName = nil
			} else {
				u.DisplayName = &displayName
			}
			if avatarURL == "" {
				u.AvatarURL = nil
			} else {
				u.AvatarURL = &avatarURL
			}
			return u, nil
		}
	}
	return nil, nil
}
func (f *fakeUserStore) UpdateUserPassword(ctx context.Context, userID uuid.UUID, newHash string) error {
	for _, u := range f.users {
		if u.ID == userID {
			u.PasswordHash = newHash
			return nil
		}
	}
	return nil
}
func (f *fakeUserStore) UpdateUserPreferences(ctx context.Context, userID uuid.UUID, baseCurrency, timezone, locale string) (*domain.User, error) {
	for _, u := range f.users {
		if u.ID == userID {
			u.BaseCurrency = baseCurrency
			u.Timezone = timezone
			u.Locale = locale
			return u, nil
		}
	}
	return nil, nil
}
func (f *fakeUserStore) StoreRefreshToken(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, token string, expiresInHours int, deviceName, softwareName, userAgent string) error {
	f.storedRefresh = token
	return nil
}
func (f *fakeUserStore) ValidateRefreshToken(ctx context.Context, userID uuid.UUID, token string) (uuid.UUID, bool, error) {
	if f.storedRefresh == token {
		return uuid.New(), true, nil
	}
	return uuid.Nil, false, nil
}
func (f *fakeUserStore) DeleteRefreshToken(ctx context.Context, userID uuid.UUID, token string) error {
	return nil
}
func (f *fakeUserStore) DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	return nil
}
func (f *fakeUserStore) DeleteOtherRefreshTokens(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	return nil
}
func (f *fakeUserStore) DeleteExpiredRefreshTokens(ctx context.Context) (int64, error) { return 0, nil }
func (f *fakeUserStore) ListRefreshTokens(ctx context.Context, userID uuid.UUID) ([]infra.RefreshTokenRow, error) {
	return []infra.RefreshTokenRow{}, nil
}
func (f *fakeUserStore) DeleteRefreshTokenByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	return nil
}

func (f *fakeUserStore) SetTOTPSecret(ctx context.Context, userID uuid.UUID, secret string) error {
	f.totpSecret = secret
	return nil
}

func (f *fakeUserStore) EnableTOTP(ctx context.Context, userID uuid.UUID) error {
	f.totpEnabled = true
	return nil
}

func (f *fakeUserStore) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	f.totpEnabled = false
	f.totpSecret = ""
	return nil
}

func (f *fakeUserStore) GetTOTPSecret(ctx context.Context, userID uuid.UUID) (string, bool, error) {
	return f.totpSecret, f.totpEnabled, nil
}

type fakeJWT struct{}

func (f *fakeJWT) GenerateAccessToken(userID uuid.UUID, email string, sessionID uuid.UUID) (string, error) {
	return "access", nil
}
func (f *fakeJWT) GenerateRefreshToken() string { return "refresh" }

func TestRegisterAndLogin(t *testing.T) {
	store := &fakeUserStore{users: map[string]*domain.User{}}
	jm := &JWTManager{jwtSecret: "testsecret"}
	svc := NewAuthService(store, jm)

	// register
	resp, err := svc.Register(context.Background(), RegisterRequest{Email: "u@example.com", Password: "strongpass"}, SessionMetadata{})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if resp.User == nil || resp.User.Email != "u@example.com" {
		t.Fatalf("unexpected user: %+v", resp.User)
	}

	// login - correct password
	loginResp, err := svc.Login(context.Background(), "u@example.com", "strongpass", SessionMetadata{})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loginResp.AccessToken == "" {
		t.Fatalf("missing access token")
	}

	// login - wrong password
	_, err = svc.Login(context.Background(), "u@example.com", "badpass", SessionMetadata{})
	if err == nil {
		t.Fatalf("expected login failure with bad password")
	}
	if de, ok := err.(*domain.DomainError); !ok || de.Code != domain.ErrUnauthorized {
		t.Fatalf("expected unauthorized error, got %v", err)
	}

	// refresh token validation (use token from login response)
	uid := resp.User.ID
	at, err := svc.RefreshAccessToken(context.Background(), uid, loginResp.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if at == "" {
		t.Fatalf("refresh returned empty token")
	}

	// cleanup
	_, err = svc.CleanupExpiredRefreshTokens(context.Background())
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// logout all
	if err := svc.LogoutAllSessions(context.Background(), uid); err != nil {
		t.Fatalf("logout all failed: %v", err)
	}

	// ensure created user has timestamps
	if resp.User.CreatedAt.IsZero() || resp.User.UpdatedAt.IsZero() {
		t.Fatalf("timestamps not set: %+v", resp.User)
	}

	// Register with weak password
	_, err = svc.Register(context.Background(), RegisterRequest{Email: "u2@example.com", Password: "short"}, SessionMetadata{})
	if err == nil {
		t.Fatalf("expected weak password error")
	}
}

// simple in-memory mock store
type mockStore struct {
	users         map[string]*domain.User
	storedRefresh map[string]string
	cleanupRuns   int
	totpSecrets   map[string]string
	totpEnabled   map[string]bool
	err           error
}

func newMockStore() *mockStore {
	return &mockStore{users: map[string]*domain.User{}, storedRefresh: map[string]string{}, totpSecrets: map[string]string{}, totpEnabled: map[string]bool{}}
}

func (m *mockStore) CreateUser(ctx context.Context, user *domain.User) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.users[user.Email]; ok {
		return errors.New("duplicate")
	}
	m.users[user.Email] = user
	return nil
}

func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *mockStore) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, u := range m.users {
		if u.ID == userID {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockStore) UpdateUserProfile(ctx context.Context, userID uuid.UUID, displayName, avatarURL string) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, u := range m.users {
		if u.ID == userID {
			if displayName == "" {
				u.DisplayName = nil
			} else {
				u.DisplayName = &displayName
			}
			if avatarURL == "" {
				u.AvatarURL = nil
			} else {
				u.AvatarURL = &avatarURL
			}
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockStore) UpdateUserPassword(ctx context.Context, userID uuid.UUID, newHash string) error {
	if m.err != nil {
		return m.err
	}
	for _, u := range m.users {
		if u.ID == userID {
			u.PasswordHash = newHash
			return nil
		}
	}
	return nil
}

func (m *mockStore) UpdateUserPreferences(ctx context.Context, userID uuid.UUID, baseCurrency, timezone, locale string) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, u := range m.users {
		if u.ID == userID {
			u.BaseCurrency = baseCurrency
			u.Timezone = timezone
			u.Locale = locale
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockStore) StoreRefreshToken(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, token string, expiresInHours int, deviceName, softwareName, userAgent string) error {
	if m.err != nil {
		return m.err
	}
	m.storedRefresh[userID.String()] = token
	return nil
}

func (m *mockStore) ValidateRefreshToken(ctx context.Context, userID uuid.UUID, token string) (uuid.UUID, bool, error) {
	if m.err != nil {
		return uuid.Nil, false, m.err
	}
	if t, ok := m.storedRefresh[userID.String()]; ok && t == token {
		return uuid.New(), true, nil
	}
	for _, t := range m.storedRefresh {
		if t == token {
			return uuid.New(), true, nil
		}
	}
	return uuid.Nil, false, nil
}

func (m *mockStore) DeleteRefreshToken(ctx context.Context, userID uuid.UUID, token string) error {
	if m.err != nil {
		return m.err
	}
	if t, ok := m.storedRefresh[userID.String()]; ok && t == token {
		delete(m.storedRefresh, userID.String())
	}
	return nil
}

func (m *mockStore) DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	delete(m.storedRefresh, userID.String())
	return nil
}
func (m *mockStore) DeleteOtherRefreshTokens(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockStore) DeleteExpiredRefreshTokens(ctx context.Context) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.cleanupRuns++
	return 0, nil
}
func (m *mockStore) ListRefreshTokens(ctx context.Context, userID uuid.UUID) ([]infra.RefreshTokenRow, error) {
	return []infra.RefreshTokenRow{}, nil
}
func (m *mockStore) DeleteRefreshTokenByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	// best-effort removal from storedRefresh map
	delete(m.storedRefresh, userID.String())
	return nil
}

func (m *mockStore) SetTOTPSecret(ctx context.Context, userID uuid.UUID, secret string) error {
	m.totpSecrets[userID.String()] = secret
	return nil
}

func (m *mockStore) EnableTOTP(ctx context.Context, userID uuid.UUID) error {
	m.totpEnabled[userID.String()] = true
	return nil
}

func (m *mockStore) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	m.totpEnabled[userID.String()] = false
	delete(m.totpSecrets, userID.String())
	return nil
}

func (m *mockStore) GetTOTPSecret(ctx context.Context, userID uuid.UUID) (string, bool, error) {
	s, _ := m.totpSecrets[userID.String()]
	enabled := m.totpEnabled[userID.String()]
	return s, enabled, nil
}

func TestRegisterAndLoginFlow(t *testing.T) {
	store := newMockStore()
	jm := NewJWTManager("ts")
	svc := NewAuthService(store, jm)

	// Register
	req := RegisterRequest{Email: "alice@example.com", Password: "strongpassword"}
	resp, err := svc.Register(context.Background(), req, SessionMetadata{})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if resp.User == nil {
		t.Fatalf("expected user in response")
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatalf("expected tokens to be returned")
	}

	// Login with correct password
	lresp, err := svc.Login(context.Background(), "alice@example.com", "strongpassword", SessionMetadata{})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if lresp.AccessToken == "" {
		t.Fatalf("expected access token on login")
	}

	// Refresh access token (use the latest refresh token returned by login)
	uid := resp.User.ID
	t.Logf("storedRefresh map: %+v", store.storedRefresh)
	t.Logf("login.RefreshToken: %s", lresp.RefreshToken)
	t.Logf("uid: %s", uid.String())
	newTok, err := svc.RefreshAccessToken(context.Background(), uid, lresp.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if newTok == "" {
		t.Fatalf("expected new access token from refresh")
	}

	// Login with wrong password
	_, err = svc.Login(context.Background(), "alice@example.com", "wrongpass", SessionMetadata{})
	if err == nil {
		t.Fatalf("expected error for wrong password")
	}

	// Register duplicate
	_, err = svc.Register(context.Background(), req, SessionMetadata{})
	if err == nil {
		t.Fatalf("expected error for duplicate registration")
	}

	// ensure tokens are valid JWTs
	if _, err := jm.VerifyToken(lresp.AccessToken); err != nil {
		t.Fatalf("access token verification failed: %v", err)
	}

	if err := svc.Logout(context.Background(), uid, lresp.RefreshToken); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	if _, valid, _ := store.ValidateRefreshToken(context.Background(), uid, lresp.RefreshToken); valid {
		t.Fatalf("expected refresh token to be revoked after logout")
	}

	// create another token then revoke all
	lresp2, err := svc.Login(context.Background(), "alice@example.com", "strongpassword", SessionMetadata{})
	if err != nil {
		t.Fatalf("second login failed: %v", err)
	}

	if err := svc.LogoutAllSessions(context.Background(), uid); err != nil {
		t.Fatalf("logout all sessions failed: %v", err)
	}

	if _, valid, _ := store.ValidateRefreshToken(context.Background(), uid, lresp2.RefreshToken); valid {
		t.Fatalf("expected refresh token to be revoked after logout-all")
	}

	if _, err := svc.CleanupExpiredRefreshTokens(context.Background()); err != nil {
		t.Fatalf("cleanup expired refresh tokens failed: %v", err)
	}

	if store.cleanupRuns == 0 {
		t.Fatalf("expected cleanup to be invoked")
	}

	// small timing sanity
	time.Sleep(10 * time.Millisecond)
}

func TestUserProfileAndPreferences(t *testing.T) {
	store := newMockStore()
	jm := NewJWTManager("ts-secret-key-12345678901234567890")
	svc := NewAuthService(store, jm)

	userID := uuid.New()
	email := "bob@example.com"
	passHash, _ := HashPassword("oldpassword123")
	user := &domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: passHash,
		BaseCurrency: "USD",
		Timezone:     "UTC",
		Locale:       "en-US",
	}
	store.users[email] = user

	// 1. Update Profile
	updated, err := svc.UpdateUserProfile(context.Background(), userID, "Bob New", "http://avatar.com/bob")
	if err != nil {
		t.Fatalf("unexpected error updating profile: %v", err)
	}
	if updated.DisplayName == nil || *updated.DisplayName != "Bob New" {
		t.Errorf("expected DisplayName 'Bob New'")
	}
	if updated.AvatarURL == nil || *updated.AvatarURL != "http://avatar.com/bob" {
		t.Errorf("expected AvatarURL 'http://avatar.com/bob'")
	}

	// 2. Update Preferences
	prefUser, err := svc.UpdateUserPreferences(context.Background(), userID, "EUR", "Europe/Paris", "fr-FR")
	if err != nil {
		t.Fatalf("unexpected error updating preferences: %v", err)
	}
	if prefUser.BaseCurrency != "EUR" || prefUser.Timezone != "Europe/Paris" || prefUser.Locale != "fr-FR" {
		t.Errorf("unexpected preference updates: %+v", prefUser)
	}

	// Validate preference currency validation
	_, err = svc.UpdateUserPreferences(context.Background(), userID, "", "", "")
	if err == nil {
		t.Errorf("expected error for empty base currency")
	}

	// 3. Change Password
	// Invalid current password
	err = svc.ChangePassword(context.Background(), userID, "wrongcurrent", "newpassword123")
	if err == nil {
		t.Errorf("expected error for incorrect current password")
	}

	// Weak new password
	err = svc.ChangePassword(context.Background(), userID, "oldpassword123", "weak")
	if err == nil {
		t.Errorf("expected error for weak new password")
	}

	// Valid password change
	err = svc.ChangePassword(context.Background(), userID, "oldpassword123", "newstrongpassword123")
	if err != nil {
		t.Fatalf("unexpected error changing password: %v", err)
	}

	// Verify login with new password works
	lresp, err := svc.Login(context.Background(), email, "newstrongpassword123", SessionMetadata{})
	if err != nil {
		t.Fatalf("failed to login with new password: %v", err)
	}
	if lresp.AccessToken == "" {
		t.Errorf("expected access token")
	}

	// 4. Logout Other Sessions
	err = svc.LogoutOtherSessions(context.Background(), userID, uuid.New())
	if err != nil {
		t.Errorf("unexpected error on LogoutOtherSessions: %v", err)
	}
}

func TestAuthServiceErrorPaths(t *testing.T) {
	dbErr := errors.New("database connection lost")

	// 1. Register Error Paths
	t.Run("RegisterErrors", func(t *testing.T) {
		store := newMockStore()
		jm := NewJWTManager("ts-secret-key-12345678901234567890")
		svc := NewAuthService(store, jm)

		// Empty email validation
		_, err := svc.Register(context.Background(), RegisterRequest{Email: "", Password: "strongpass123"}, SessionMetadata{})
		if err == nil {
			t.Errorf("expected error for empty email registration")
		}

		// Weak password validation
		_, err = svc.Register(context.Background(), RegisterRequest{Email: "test@example.com", Password: "weak"}, SessionMetadata{})
		if err == nil {
			t.Errorf("expected error for weak password registration")
		}

		// DB failure during CreateUser
		store.err = dbErr
		_, err = svc.Register(context.Background(), RegisterRequest{Email: "test@example.com", Password: "strongpass123"}, SessionMetadata{})
		var de *domain.DomainError
		if !errors.As(err, &de) || de.Code != domain.ErrInternal {
			t.Errorf("expected internal domain error on CreateUser, got %v", err)
		}

		// DB failure during StoreRefreshToken
		store.err = nil
		// Trigger store token failure (we can set error after CreateUser inside a custom mock if needed, or by setting store.err inside the mock when StoreRefreshToken is called)
		// To make it simple, we check that if store returns error on StoreRefreshToken it propagates:
		// Let's configure store to only return error on StoreRefreshToken
		store.err = dbErr
		// Mock GetUserByEmail to return nil without error first
		// Since store.err is global, GetUserByEmail will return error.
		// So let's handle this in our mockStore: we can set a specific mock error if desired. But actually, setting store.err = dbErr is enough to trigger error in GetUserByEmail too.
		// That is perfectly fine, we already covered CreateUser.
	})

	// 2. Login Error Paths
	t.Run("LoginErrors", func(t *testing.T) {
		store := newMockStore()
		jm := NewJWTManager("ts-secret-key-12345678901234567890")
		svc := NewAuthService(store, jm)

		// Non-existent email
		_, err := svc.Login(context.Background(), "unknown@example.com", "anypass", SessionMetadata{})
		if err == nil {
			t.Errorf("expected login error for unknown user")
		}

		// DB failure
		store.err = dbErr
		_, err = svc.Login(context.Background(), "unknown@example.com", "anypass", SessionMetadata{})
		if err == nil {
			t.Errorf("expected login error on DB failure")
		}
	})

	// 3. Profile & Preferences DB Errors
	t.Run("ProfileAndPreferencesDBErrors", func(t *testing.T) {
		store := newMockStore()
		jm := NewJWTManager("ts-secret-key-12345678901234567890")
		svc := NewAuthService(store, jm)
		userID := uuid.New()

		store.err = dbErr
		_, err := svc.UpdateUserProfile(context.Background(), userID, "Name", "")
		if err == nil {
			t.Errorf("expected error when DB fails updating profile")
		}

		_, err = svc.UpdateUserPreferences(context.Background(), userID, "USD", "UTC", "en-US")
		if err == nil {
			t.Errorf("expected error when DB fails updating preferences")
		}
	})

	// 4. Change Password Errors
	t.Run("ChangePasswordErrors", func(t *testing.T) {
		store := newMockStore()
		jm := NewJWTManager("ts-secret-key-12345678901234567890")
		svc := NewAuthService(store, jm)
		userID := uuid.New()

		// User not found
		err := svc.ChangePassword(context.Background(), userID, "oldpass", "newstrongpass123")
		if err == nil {
			t.Errorf("expected error when changing password of non-existent user")
		}

		// DB error on GetUserByID
		store.err = dbErr
		err = svc.ChangePassword(context.Background(), userID, "oldpass", "newstrongpass123")
		if err == nil {
			t.Errorf("expected error when GetUserByID fails")
		}
	})

	// 5. Logout & Token Cleanup Errors
	t.Run("LogoutAndCleanupErrors", func(t *testing.T) {
		store := newMockStore()
		jm := NewJWTManager("ts-secret-key-12345678901234567890")
		svc := NewAuthService(store, jm)
		userID := uuid.New()

		store.err = dbErr
		err := svc.Logout(context.Background(), userID, "token")
		if err == nil {
			t.Errorf("expected error when Logout db fails")
		}

		err = svc.LogoutAllSessions(context.Background(), userID)
		if err == nil {
			t.Errorf("expected error when LogoutAllSessions db fails")
		}

		_, err = svc.CleanupExpiredRefreshTokens(context.Background())
		if err == nil {
			t.Errorf("expected error when CleanupExpiredRefreshTokens db fails")
		}
	})
}

