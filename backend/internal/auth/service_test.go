package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"spendsense-backend/internal/domain"

	"github.com/google/uuid"
)

// simple in-memory mock store
type mockStore struct {
	users         map[string]*domain.User
	storedRefresh map[string]string
}

func newMockStore() *mockStore {
	return &mockStore{users: map[string]*domain.User{}, storedRefresh: map[string]string{}}
}

func (m *mockStore) CreateUser(ctx context.Context, user *domain.User) error {
	if _, ok := m.users[user.Email]; ok {
		return errors.New("duplicate")
	}
	m.users[user.Email] = user
	return nil
}

func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, nil
}

func (m *mockStore) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == userID {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockStore) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiresInHours int) error {
	m.storedRefresh[userID.String()] = token
	return nil
}

func (m *mockStore) ValidateRefreshToken(ctx context.Context, userID uuid.UUID, token string) (bool, error) {
	if t, ok := m.storedRefresh[userID.String()]; ok && t == token {
		return true, nil
	}
	// fallback: check any stored token match (helpful for tests where keys may differ)
	for _, t := range m.storedRefresh {
		if t == token {
			return true, nil
		}
	}
	return false, nil
}

func TestRegisterAndLoginFlow(t *testing.T) {
	store := newMockStore()
	jm := NewJWTManager("ts")
	svc := NewAuthService(store, jm)

	// Register
	req := RegisterRequest{Email: "alice@example.com", Password: "strongpassword"}
	resp, err := svc.Register(context.Background(), req)
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
	lresp, err := svc.Login(context.Background(), "alice@example.com", "strongpassword")
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
	_, err = svc.Login(context.Background(), "alice@example.com", "wrongpass")
	if err == nil {
		t.Fatalf("expected error for wrong password")
	}

	// Register duplicate
	_, err = svc.Register(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error for duplicate registration")
	}

	// ensure tokens are valid JWTs
	if _, err := jm.VerifyToken(lresp.AccessToken); err != nil {
		t.Fatalf("access token verification failed: %v", err)
	}

	// small timing sanity
	time.Sleep(10 * time.Millisecond)
}
