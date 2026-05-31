package infra

import (
	"context"
	"database/sql"
	"log"
	"spendsense-backend/internal/domain"
	"strings"

	"github.com/google/uuid"
)

func (db *Database) CreateUser(ctx context.Context, user *domain.User) error {
	_, err := db.pool.Exec(ctx, `
	INSERT INTO users (id, email, password_hash, display_name, base_currency, timezone, locale)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.Email, user.PasswordHash, user.DisplayName, user.BaseCurrency, user.Timezone, user.Locale)

	return err
}

func (db *Database) GetUserByID(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	row := db.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, avatar_url, base_currency, timezone, locale, totp_secret, COALESCE(totp_enabled, FALSE), totp_confirmed_at, created_at, updated_at
		FROM users WHERE id = $1
	`, userID)

	user := &domain.User{}
	var displayName sql.NullString
	var avatarURL sql.NullString
	var totpSecret sql.NullString
	var totpEnabled bool
	var totpConfirmed sql.NullTime
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &displayName, &avatarURL,
		&user.BaseCurrency, &user.Timezone, &user.Locale, &totpSecret, &totpEnabled, &totpConfirmed, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isMissingColumnError(err) {
			// fallback to older schema
			row2 := db.pool.QueryRow(ctx, `
				SELECT id, email, password_hash, display_name, avatar_url, base_currency, timezone, locale, created_at, updated_at
				FROM users WHERE id = $1
			`, userID)
			err2 := row2.Scan(&user.ID, &user.Email, &user.PasswordHash, &displayName, &avatarURL,
				&user.BaseCurrency, &user.Timezone, &user.Locale, &user.CreatedAt, &user.UpdatedAt)
			if err2 != nil {
				return nil, err2
			}
			if displayName.Valid {
				user.DisplayName = &displayName.String
			}
			if avatarURL.Valid {
				user.AvatarURL = &avatarURL.String
			}
			return user, nil
		}
		return nil, err
	}
	if displayName.Valid {
		user.DisplayName = &displayName.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if totpSecret.Valid {
		user.TOTPSecret = &totpSecret.String
	}
	user.TOTPEnabled = totpEnabled
	if totpConfirmed.Valid {
		user.TOTPConfirmedAt = &totpConfirmed.Time
	}
	return user, nil
}

func (db *Database) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := db.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, avatar_url, base_currency, timezone, locale, totp_secret, COALESCE(totp_enabled, FALSE), totp_confirmed_at, created_at, updated_at
		FROM users WHERE email = $1
	`, email)

	user := &domain.User{}
	var displayName sql.NullString
	var avatarURL sql.NullString
	var totpSecret sql.NullString
	var totpEnabled bool
	var totpConfirmed sql.NullTime
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &displayName, &avatarURL,
		&user.BaseCurrency, &user.Timezone, &user.Locale, &totpSecret, &totpEnabled, &totpConfirmed, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if isMissingColumnError(err) {
			row2 := db.pool.QueryRow(ctx, `
				SELECT id, email, password_hash, display_name, avatar_url, base_currency, timezone, locale, created_at, updated_at
				FROM users WHERE email = $1
			`, email)
			err2 := row2.Scan(&user.ID, &user.Email, &user.PasswordHash, &displayName, &avatarURL,
				&user.BaseCurrency, &user.Timezone, &user.Locale, &user.CreatedAt, &user.UpdatedAt)
			if err2 != nil {
				return nil, err2
			}
			if displayName.Valid {
				user.DisplayName = &displayName.String
			}
			if avatarURL.Valid {
				user.AvatarURL = &avatarURL.String
			}
			return user, nil
		}
		return nil, err
	}
	if displayName.Valid {
		user.DisplayName = &displayName.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if totpSecret.Valid {
		user.TOTPSecret = &totpSecret.String
	}
	user.TOTPEnabled = totpEnabled
	if totpConfirmed.Valid {
		user.TOTPConfirmedAt = &totpConfirmed.Time
	}
	return user, nil
}

// SetTOTPSecret stores the base32 secret for user's TOTP
func (db *Database) SetTOTPSecret(ctx context.Context, userID uuid.UUID, secret string) error {
	tag, err := db.pool.Exec(ctx, `
		UPDATE users SET totp_secret = $2, updated_at = NOW() WHERE id = $1
	`, userID, secret)
	if err != nil {
		if isMissingColumnError(err) {
			log.Printf("SetTOTPSecret: schema missing totp columns, skipping persist for user %s", userID.String())
			return nil
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		log.Printf("SetTOTPSecret: no rows updated for user %s", userID.String())
	}
	return nil
}

// EnableTOTP marks TOTP enabled for user
func (db *Database) EnableTOTP(ctx context.Context, userID uuid.UUID) error {
	tag, err := db.pool.Exec(ctx, `
		UPDATE users SET totp_enabled = TRUE, totp_confirmed_at = NOW(), updated_at = NOW() WHERE id = $1
	`, userID)
	if err != nil {
		if isMissingColumnError(err) {
			log.Printf("EnableTOTP: schema missing totp columns, skipping enable for user %s", userID.String())
			return nil
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		log.Printf("EnableTOTP: no rows updated for user %s", userID.String())
	}
	return nil
}

// DisableTOTP disables TOTP and clears the secret
func (db *Database) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	tag, err := db.pool.Exec(ctx, `
		UPDATE users SET totp_enabled = FALSE, totp_secret = NULL, totp_confirmed_at = NULL, updated_at = NOW() WHERE id = $1
	`, userID)
	if err != nil {
		if isMissingColumnError(err) {
			log.Printf("DisableTOTP: schema missing totp columns, skipping disable for user %s", userID.String())
			return nil
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		log.Printf("DisableTOTP: no rows updated for user %s", userID.String())
	}
	return nil
}

// GetTOTPSecret returns stored secret and whether totp is enabled
func (db *Database) GetTOTPSecret(ctx context.Context, userID uuid.UUID) (string, bool, error) {
	row := db.pool.QueryRow(ctx, `SELECT COALESCE(totp_secret, ''), COALESCE(totp_enabled, FALSE) FROM users WHERE id = $1`, userID)
	var secret string
	var enabled bool
	if err := row.Scan(&secret, &enabled); err != nil {
		if isMissingColumnError(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return secret, enabled, nil
}

func (db *Database) UpdateUserProfile(ctx context.Context, userID uuid.UUID, displayName, avatarURL string) (*domain.User, error) {
	var displayNameValue any
	if trimmed := strings.TrimSpace(displayName); trimmed != "" {
		displayNameValue = trimmed
	}

	var avatarURLValue any
	if trimmed := strings.TrimSpace(avatarURL); trimmed != "" {
		avatarURLValue = trimmed
	}

	row := db.pool.QueryRow(ctx, `
		UPDATE users
		SET display_name = $2,
		    avatar_url = $3,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, password_hash, display_name, avatar_url, base_currency, timezone, locale, created_at, updated_at
	`, userID, displayNameValue, avatarURLValue)

	user := &domain.User{}
	var displayNameResult sql.NullString
	var avatarURLResult sql.NullString
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &displayNameResult, &avatarURLResult,
		&user.BaseCurrency, &user.Timezone, &user.Locale, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if displayNameResult.Valid {
		user.DisplayName = &displayNameResult.String
	}
	if avatarURLResult.Valid {
		user.AvatarURL = &avatarURLResult.String
	}
	return user, nil
}

func (db *Database) UpdateUserPassword(ctx context.Context, userID uuid.UUID, newHash string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2,
			updated_at = NOW()
		WHERE id = $1
	`, userID, newHash)

	return err
}

func (db *Database) UpdateUserPreferences(ctx context.Context, userID uuid.UUID, baseCurrency, timezone, locale string) (*domain.User, error) {
	row := db.pool.QueryRow(ctx, `
		UPDATE users
		SET base_currency = $2,
		    timezone = $3,
		    locale = $4,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, password_hash, display_name, avatar_url, base_currency, timezone, locale, created_at, updated_at
	`, userID, strings.ToUpper(strings.TrimSpace(baseCurrency)), strings.TrimSpace(timezone), strings.TrimSpace(locale))

	user := &domain.User{}
	var displayName sql.NullString
	var avatarURL sql.NullString
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &displayName, &avatarURL,
		&user.BaseCurrency, &user.Timezone, &user.Locale, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if displayName.Valid {
		user.DisplayName = &displayName.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	return user, nil
}
