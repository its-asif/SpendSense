package infra

import (
	"context"
	"database/sql"
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
		SELECT id, email, password_hash, display_name, avatar_url, base_currency, timezone, locale, created_at, updated_at
		FROM users WHERE id = $1
	`, userID)

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

func (db *Database) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := db.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, avatar_url, base_currency, timezone, locale, created_at, updated_at
		FROM users WHERE email = $1
	`, email)

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
