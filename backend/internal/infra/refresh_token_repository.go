package infra

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// StoreRefreshToken hashes and stores a refresh token with an expiry time (hours)
func (db *Database) StoreRefreshToken(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, token string, expiresInHours int, device, ipAddress, userAgent string) error {
	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(time.Duration(expiresInHours) * time.Hour)

	_, err := db.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, device, ip_address, user_agent, last_seen_at, revoked)
		VALUES ($1, $2, $3, $4, NOW(), $5, $6, $7, NOW(), FALSE)
	`, sessionID, userID, tokenHash, expiresAt, device, ipAddress, userAgent)
	if err == nil || !isMissingColumnError(err) {
		return err
	}

	_, err = db.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, sessionID, userID, tokenHash, expiresAt)
	return err
}

// ValidateRefreshToken returns the session id if the token exists and is not expired
func (db *Database) ValidateRefreshToken(ctx context.Context, userID uuid.UUID, token string) (uuid.UUID, bool, error) {
	tokenHash := hashToken(token)
	row := db.pool.QueryRow(ctx, `
		SELECT id FROM refresh_tokens
		WHERE user_id = $1 AND token_hash = $2 AND expires_at > NOW() AND COALESCE(revoked, FALSE) = FALSE
	`, userID, tokenHash)

	var sessionID uuid.UUID
	if err := row.Scan(&sessionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, false, nil
		}
		if isMissingColumnError(err) {
			fallback := db.pool.QueryRow(ctx, `
				SELECT id FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2 AND expires_at > NOW()
			`, userID, tokenHash)
			if scanErr := fallback.Scan(&sessionID); scanErr != nil {
				if errors.Is(scanErr, pgx.ErrNoRows) {
					return uuid.Nil, false, nil
				}
				return uuid.Nil, false, scanErr
			}
			return sessionID, true, nil
		}
		return uuid.Nil, false, err
	}

	_, _ = db.pool.Exec(ctx, `
		UPDATE refresh_tokens SET last_seen_at = NOW() WHERE user_id = $1 AND id = $2
	`, userID, sessionID)

	return sessionID, true, nil
}

// DeleteRefreshToken removes a stored refresh token for a user
func (db *Database) DeleteRefreshToken(ctx context.Context, userID uuid.UUID, token string) error {
	tokenHash := hashToken(token)
	_, err := db.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked = TRUE, revoked_at = NOW(), last_seen_at = NOW()
		WHERE user_id = $1 AND token_hash = $2
	`, userID, tokenHash)
	if err != nil && isMissingColumnError(err) {
		_, err = db.pool.Exec(ctx, `
			DELETE FROM refresh_tokens WHERE user_id = $1 AND token_hash = $2
		`, userID, tokenHash)
	}
	return err
}

// DeleteAllRefreshTokens removes all stored refresh tokens for a user.
func (db *Database) DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked = TRUE, revoked_at = NOW(), last_seen_at = NOW()
		WHERE user_id = $1
	`, userID)
	if err != nil && isMissingColumnError(err) {
		_, err = db.pool.Exec(ctx, `
			DELETE FROM refresh_tokens WHERE user_id = $1
		`, userID)
	}
	return err
}

// DeleteExpiredRefreshTokens removes expired refresh tokens and returns deleted rows.
func (db *Database) DeleteExpiredRefreshTokens(ctx context.Context) (int64, error) {
	tag, err := db.pool.Exec(ctx, `
		DELETE FROM refresh_tokens WHERE expires_at <= NOW()
	`)
	if err != nil {
		return 0, err
	}

	deleted := tag.RowsAffected()
	if deleted < 0 {
		return 0, fmt.Errorf("invalid rows affected: %d", deleted)
	}

	return deleted, nil
}

type RefreshTokenRow struct {
	SessionID        uuid.UUID `json:"session_id"`
	ID               uuid.UUID `json:"id,omitempty"`
	Device           string    `json:"device"`
	IP               string    `json:"ip"`
	LastSeen         time.Time `json:"last_seen"`
	RefreshTokenHash string    `json:"refresh_token_hash"`
	Revoked          bool      `json:"revoked"`
	CreatedAt        time.Time `json:"created_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// ListRefreshTokens returns stored refresh tokens for a user
func (db *Database) ListRefreshTokens(ctx context.Context, userID uuid.UUID) ([]RefreshTokenRow, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, COALESCE(device, ''), COALESCE(ip_address, ''), COALESCE(last_seen_at, created_at), token_hash, COALESCE(revoked, FALSE), created_at, expires_at
		FROM refresh_tokens WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil && isMissingColumnError(err) {
		rows, err = db.pool.Query(ctx, `
			SELECT id, token_hash, created_at, expires_at FROM refresh_tokens WHERE user_id = $1 ORDER BY created_at DESC
		`, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var out []RefreshTokenRow
		for rows.Next() {
			var r RefreshTokenRow
			if err := rows.Scan(&r.SessionID, &r.RefreshTokenHash, &r.CreatedAt, &r.ExpiresAt); err != nil {
				return nil, err
			}
			r.ID = r.SessionID
			r.LastSeen = r.CreatedAt
			r.Revoked = false
			out = append(out, r)
		}
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RefreshTokenRow
	for rows.Next() {
		var r RefreshTokenRow
		if err := rows.Scan(&r.SessionID, &r.Device, &r.IP, &r.LastSeen, &r.RefreshTokenHash, &r.Revoked, &r.CreatedAt, &r.ExpiresAt); err != nil {
			return nil, err
		}
		r.ID = r.SessionID
		out = append(out, r)
	}
	return out, nil
}

// DeleteOtherRefreshTokens removes all other sessions for a user except the active one.
func (db *Database) DeleteOtherRefreshTokens(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked = TRUE, revoked_at = NOW(), last_seen_at = NOW()
		WHERE user_id = $1 AND id <> $2
	`, userID, sessionID)
	if err != nil && isMissingColumnError(err) {
		_, err = db.pool.Exec(ctx, `
			DELETE FROM refresh_tokens WHERE user_id = $1 AND id <> $2
		`, userID, sessionID)
	}
	return err
}

// DeleteRefreshTokenByID removes a refresh token row by its id for the given user
func (db *Database) DeleteRefreshTokenByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked = TRUE, revoked_at = NOW(), last_seen_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil && isMissingColumnError(err) {
		_, err = db.pool.Exec(ctx, `
			DELETE FROM refresh_tokens WHERE id = $1 AND user_id = $2
		`, id, userID)
	}
	return err
}

// HasActiveSession returns whether the session row still exists for the user.
func (db *Database) HasActiveSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (bool, error) {
	tag, err := db.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET last_seen_at = NOW()
		WHERE user_id = $1 AND id = $2 AND expires_at > NOW() AND COALESCE(revoked, FALSE) = FALSE
	`, userID, sessionID)
	if err != nil {
		if isMissingColumnError(err) {
			row := db.pool.QueryRow(ctx, `
				SELECT EXISTS(
					SELECT 1 FROM refresh_tokens WHERE user_id = $1 AND id = $2 AND expires_at > NOW()
				)
			`, userID, sessionID)
			var active bool
			if scanErr := row.Scan(&active); scanErr != nil {
				return false, scanErr
			}
			return active, nil
		}
		return false, err
	}

	return tag.RowsAffected() > 0, nil
}

func isMissingColumnError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "column") && strings.Contains(message, "does not exist")
}
