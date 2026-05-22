package infra

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// StoreRefreshToken hashes and stores a refresh token with an expiry time (hours)
func (db *Database) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiresInHours int) error {
	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(time.Duration(expiresInHours) * time.Hour)

	_, err := db.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, uuid.New(), userID, tokenHash, expiresAt)
	return err
}

// ValidateRefreshToken returns true if the token exists and is not expired
func (db *Database) ValidateRefreshToken(ctx context.Context, userID uuid.UUID, token string) (bool, error) {
	tokenHash := hashToken(token)
	row := db.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2 AND expires_at > NOW()
		)
	`, userID, tokenHash)

	var valid bool
	if err := row.Scan(&valid); err != nil {
		return false, err
	}
	return valid, nil
}
