package expense

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
	CreateExpense(ctx context.Context, expense *Expense) error
	GetExpenseByID(ctx context.Context, userID, expenseID uuid.UUID) (*Expense, error)
	ListExpenses(ctx context.Context, userID uuid.UUID, limit int, cursor *Cursor) ([]*Expense, *Cursor, error)
	UpdateExpense(ctx context.Context, expense *Expense) error
	SoftDeleteExpense(ctx context.Context, userID, expenseID uuid.UUID) error
}

type Repository struct {
	db *infra.Database
}

func NewRepository(db *infra.Database) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateExpense(ctx context.Context, expense *Expense) error {
	if expense == nil {
		return domain.NewDomainError(domain.ErrInternal, "expense is required", 400)
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO expenses (
			id, user_id, wallet_id, amount, currency, fx_rate_to_base, category_id,
			merchant, date, notes, is_recurring, recurring_rule, is_deleted
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, FALSE)
		RETURNING created_at, updated_at, deleted_at
	`, expense.ID, expense.UserID, expense.WalletID, expense.Amount, expense.Currency,
		expense.FXRateToBase, expense.CategoryID, expense.Merchant, expense.Date,
		expense.Notes, expense.IsRecurring, expense.RecurringRule)

	var deletedAt sql.NullTime
	if err := row.Scan(&expense.CreatedAt, &expense.UpdatedAt, &deletedAt); err != nil {
		return err
	}
	if deletedAt.Valid {
		expense.DeletedAt = &deletedAt.Time
	}

	return nil
}

func (r *Repository) GetExpenseByID(ctx context.Context, userID, expenseID uuid.UUID) (*Expense, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, wallet_id, amount, currency, fx_rate_to_base, category_id,
			merchant, date, notes, is_recurring, recurring_rule, is_deleted,
			created_at, updated_at, deleted_at
		FROM expenses
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
	`, userID, expenseID)

	expense, err := scanExpense(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.NewDomainError(domain.ErrNotFound, "expense not found", 404)
		}
		return nil, err
	}

	return expense, nil
}

func (r *Repository) ListExpenses(ctx context.Context, userID uuid.UUID, limit int, cursor *Cursor) ([]*Expense, *Cursor, error) {
	query := `
		SELECT id, user_id, wallet_id, amount, currency, fx_rate_to_base, category_id,
			merchant, date, notes, is_recurring, recurring_rule, is_deleted,
			created_at, updated_at, deleted_at
		FROM expenses
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

	expenses := make([]*Expense, 0, limit)
	for rows.Next() {
		expense, err := scanExpense(rows)
		if err != nil {
			return nil, nil, err
		}
		expenses = append(expenses, expense)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var nextCursor *Cursor
	if len(expenses) > limit {
		last := expenses[limit-1]
		nextCursor = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
		expenses = expenses[:limit]
	}

	return expenses, nextCursor, nil
}

func (r *Repository) UpdateExpense(ctx context.Context, expense *Expense) error {
	if expense == nil {
		return domain.NewDomainError(domain.ErrInternal, "expense is required", 400)
	}

	row := r.db.QueryRow(ctx, `
		UPDATE expenses
		SET wallet_id = $3,
			amount = $4,
			currency = $5,
			fx_rate_to_base = $6,
			category_id = $7,
			merchant = $8,
			date = $9,
			notes = $10,
			is_recurring = $11,
			recurring_rule = $12,
			updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
		RETURNING created_at, updated_at, deleted_at
	`, expense.UserID, expense.ID, expense.WalletID, expense.Amount, expense.Currency,
		expense.FXRateToBase, expense.CategoryID, expense.Merchant, expense.Date,
		expense.Notes, expense.IsRecurring, expense.RecurringRule)

	var deletedAt sql.NullTime
	if err := row.Scan(&expense.CreatedAt, &expense.UpdatedAt, &deletedAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "expense not found", 404)
		}
		return err
	}
	if deletedAt.Valid {
		expense.DeletedAt = &deletedAt.Time
	} else {
		expense.DeletedAt = nil
	}

	return nil
}

func (r *Repository) SoftDeleteExpense(ctx context.Context, userID, expenseID uuid.UUID) error {
	row := r.db.QueryRow(ctx, `
		UPDATE expenses
		SET is_deleted = TRUE,
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
		RETURNING id
	`, userID, expenseID)

	var id uuid.UUID
	if err := row.Scan(&id); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "expense not found", 404)
		}
		return err
	}

	return nil
}

func scanExpense(scanner interface{ Scan(...any) error }) (*Expense, error) {
	expense := &Expense{}
	var merchant sql.NullString
	var notes sql.NullString
	var recurringRule sql.NullString
	var deletedAt sql.NullTime

	err := scanner.Scan(
		&expense.ID,
		&expense.UserID,
		&expense.WalletID,
		&expense.Amount,
		&expense.Currency,
		&expense.FXRateToBase,
		&expense.CategoryID,
		&merchant,
		&expense.Date,
		&notes,
		&expense.IsRecurring,
		&recurringRule,
		&expense.IsDeleted,
		&expense.CreatedAt,
		&expense.UpdatedAt,
		&deletedAt,
	)
	if err != nil {
		return nil, err
	}

	if merchant.Valid {
		expense.Merchant = &merchant.String
	}
	if notes.Valid {
		expense.Notes = &notes.String
	}
	if recurringRule.Valid {
		expense.RecurringRule = &recurringRule.String
	}
	if deletedAt.Valid {
		expense.DeletedAt = &deletedAt.Time
	}

	return expense, nil
}
