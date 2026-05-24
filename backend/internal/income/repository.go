package income

import (
	"context"
	"database/sql"
	"fmt"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/infra"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Store interface {
	CreateIncome(ctx context.Context, income *Income) error
	GetIncomeByID(ctx context.Context, userID, incomeID uuid.UUID) (*Income, error)
	ListIncomes(ctx context.Context, userID uuid.UUID, limit int, cursor *Cursor) ([]*Income, *Cursor, error)
	UpdateIncome(ctx context.Context, income *Income) error
	SoftDeleteIncome(ctx context.Context, userID, incomeID uuid.UUID) error
}

type Repository struct {
	db *infra.Database
}

func NewRepository(db *infra.Database) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateIncome(ctx context.Context, income *Income) error {
	if income == nil {
		return domain.NewDomainError(domain.ErrInternal, "income is required", 400)
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO incomes (
			id, user_id, wallet_id, category_id, source_name, amount, currency,
			income_date, notes, is_deleted
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, FALSE)
		RETURNING created_at, updated_at, deleted_at
	`, income.ID, income.UserID, income.WalletID, income.CategoryID, income.SourceName,
		income.Amount, income.Currency, income.IncomeDate, income.Notes)

	var deletedAt sql.NullTime
	if err := row.Scan(&income.CreatedAt, &income.UpdatedAt, &deletedAt); err != nil {
		return err
	}
	if deletedAt.Valid {
		income.DeletedAt = &deletedAt.Time
	}

	return nil
}

func (r *Repository) GetIncomeByID(ctx context.Context, userID, incomeID uuid.UUID) (*Income, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, wallet_id, category_id, source_name, amount, currency,
			income_date, notes, is_deleted, created_at, updated_at, deleted_at
		FROM incomes
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
	`, userID, incomeID)

	income, err := scanIncome(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
		}
		return nil, err
	}

	return income, nil
}

func (r *Repository) ListIncomes(ctx context.Context, userID uuid.UUID, limit int, cursor *Cursor) ([]*Income, *Cursor, error) {
	query := `
		SELECT id, user_id, wallet_id, category_id, source_name, amount, currency,
			income_date, notes, is_deleted, created_at, updated_at, deleted_at
		FROM incomes
		WHERE user_id = $1 AND is_deleted = FALSE
	`

	args := []any{userID}
	if cursor != nil {
		query += " AND (created_at, id) < ($2, $3)"
		args = append(args, cursor.CreatedAt, cursor.ID)
	}

	args = append(args, limit+1)
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	incomes := make([]*Income, 0, limit)
	for rows.Next() {
		income, err := scanIncome(rows)
		if err != nil {
			return nil, nil, err
		}
		incomes = append(incomes, income)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var nextCursor *Cursor
	if len(incomes) > limit {
		last := incomes[limit-1]
		nextCursor = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
		incomes = incomes[:limit]
	}

	return incomes, nextCursor, nil
}

func (r *Repository) UpdateIncome(ctx context.Context, income *Income) error {
	if income == nil {
		return domain.NewDomainError(domain.ErrInternal, "income is required", 400)
	}

	row := r.db.QueryRow(ctx, `
		UPDATE incomes
		SET wallet_id = $3,
			category_id = $4,
			source_name = $5,
			amount = $6,
			currency = $7,
			income_date = $8,
			notes = $9,
			updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
		RETURNING created_at, updated_at, deleted_at
	`, income.UserID, income.ID, income.WalletID, income.CategoryID, income.SourceName,
		income.Amount, income.Currency, income.IncomeDate, income.Notes)

	var deletedAt sql.NullTime
	if err := row.Scan(&income.CreatedAt, &income.UpdatedAt, &deletedAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
		}
		return err
	}
	if deletedAt.Valid {
		income.DeletedAt = &deletedAt.Time
	} else {
		income.DeletedAt = nil
	}

	return nil
}

func (r *Repository) SoftDeleteIncome(ctx context.Context, userID, incomeID uuid.UUID) error {
	row := r.db.QueryRow(ctx, `
		UPDATE incomes
		SET is_deleted = TRUE,
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
		RETURNING id
	`, userID, incomeID)

	var id uuid.UUID
	if err := row.Scan(&id); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
		}
		return err
	}

	return nil
}

func scanIncome(scanner interface{ Scan(...any) error }) (*Income, error) {
	income := &Income{}
	var categoryID sql.NullString
	var notes sql.NullString
	var deletedAt sql.NullTime

	err := scanner.Scan(
		&income.ID,
		&income.UserID,
		&income.WalletID,
		&categoryID,
		&income.SourceName,
		&income.Amount,
		&income.Currency,
		&income.IncomeDate,
		&notes,
		&income.IsDeleted,
		&income.CreatedAt,
		&income.UpdatedAt,
		&deletedAt,
	)
	if err != nil {
		return nil, err
	}

	if categoryID.Valid {
		id, err := uuid.Parse(categoryID.String)
		if err != nil {
			return nil, err
		}
		income.CategoryID = &id
	}
	if notes.Valid {
		income.Notes = &notes.String
	}
	if deletedAt.Valid {
		income.DeletedAt = &deletedAt.Time
	}

	return income, nil
}
