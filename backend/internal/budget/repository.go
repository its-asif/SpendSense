package budget

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/infra"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db *infra.Database }

func NewRepository(db *infra.Database) *Repository { return &Repository{db: db} }

func (r *Repository) CategoryAccessible(ctx context.Context, userID, categoryID uuid.UUID) (bool, error) {
	row := r.db.QueryRow(ctx, `
		SELECT 1 FROM categories
		WHERE id = $1 AND kind = 'EXPENSE' AND (user_id = $2 OR is_default = TRUE)
		LIMIT 1
	`, categoryID, userID)
	var marker int
	if err := row.Scan(&marker); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) HasMonthlyBudgetForCategory(ctx context.Context, userID, categoryID uuid.UUID, excludeID *uuid.UUID) (bool, error) {
	query := `
		SELECT 1 FROM budgets
		WHERE user_id = $1 AND category_id = $2 AND UPPER(period) = 'MONTHLY'
	`
	args := []any{userID, categoryID}
	if excludeID != nil {
		query += ` AND id <> $3`
		args = append(args, *excludeID)
	}
	query += ` LIMIT 1`

	row := r.db.QueryRow(ctx, query, args...)
	var marker int
	if err := row.Scan(&marker); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) Create(ctx context.Context, b *Budget) error {
	row := r.db.QueryRow(ctx, `
		INSERT INTO budgets (
			id, user_id, category_id, amount, currency, period, start_date, rollover_enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`, b.ID, b.UserID, b.CategoryID, b.Amount, b.Currency, strings.ToUpper(b.Period), b.StartDate, b.RolloverEnabled)
	return row.Scan(&b.CreatedAt, &b.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, userID, id uuid.UUID) (*Budget, error) {
	return r.scanBudget(r.db.QueryRow(ctx, `
		SELECT
			b.id, b.user_id, b.category_id,
			COALESCE(c.name, 'Uncategorized'),
			c.icon, c.color,
			b.amount, b.currency, b.period, b.start_date, b.rollover_enabled,
			b.created_at, b.updated_at
		FROM budgets b
		LEFT JOIN categories c ON c.id = b.category_id
		WHERE b.id = $1 AND b.user_id = $2
	`, id, userID))
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID, period string) ([]*Budget, error) {
	query := `
		SELECT
			b.id, b.user_id, b.category_id,
			COALESCE(c.name, 'Uncategorized'),
			c.icon, c.color,
			b.amount, b.currency, b.period, b.start_date, b.rollover_enabled,
			b.created_at, b.updated_at
		FROM budgets b
		LEFT JOIN categories c ON c.id = b.category_id
		WHERE b.user_id = $1
	`
	args := []any{userID}
	if period != "" {
		query += ` AND UPPER(b.period) = UPPER($2)`
		args = append(args, period)
	}
	query += ` ORDER BY COALESCE(c.name, 'Uncategorized') ASC, b.created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*Budget, 0)
	for rows.Next() {
		b, err := r.scanBudgetRows(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

func (r *Repository) Update(ctx context.Context, b *Budget) error {
	row := r.db.QueryRow(ctx, `
		UPDATE budgets
		SET category_id = $3,
		    amount = $4,
		    currency = $5,
		    period = $6,
		    start_date = $7,
		    rollover_enabled = $8,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2
		RETURNING created_at, updated_at
	`, b.ID, b.UserID, b.CategoryID, b.Amount, b.Currency, strings.ToUpper(b.Period), b.StartDate, b.RolloverEnabled)
	if err := row.Scan(&b.CreatedAt, &b.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "budget not found", 404)
		}
		return err
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM budgets WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrNotFound, "budget not found", 404)
	}
	return nil
}

func (r *Repository) scanBudget(row pgx.Row) (*Budget, error) {
	b := &Budget{}
	var categoryID uuid.UUID
	var icon sql.NullString
	var color sql.NullString
	var startDate time.Time

	err := row.Scan(
		&b.ID, &b.UserID, &categoryID,
		&b.CategoryName, &icon, &color,
		&b.Amount, &b.Currency, &b.Period, &startDate, &b.RolloverEnabled,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.NewDomainError(domain.ErrNotFound, "budget not found", 404)
		}
		return nil, err
	}

	b.CategoryID = categoryID
	b.StartDate = startDate
	if icon.Valid {
		value := icon.String
		b.CategoryIcon = &value
	}
	if color.Valid {
		value := color.String
		b.CategoryColor = &value
	}
	b.Period = strings.ToUpper(strings.TrimSpace(b.Period))
	return b, nil
}

func (r *Repository) scanBudgetRows(rows pgx.Rows) (*Budget, error) {
	b := &Budget{}
	var categoryID uuid.UUID
	var icon sql.NullString
	var color sql.NullString
	var startDate time.Time

	if err := rows.Scan(
		&b.ID, &b.UserID, &categoryID,
		&b.CategoryName, &icon, &color,
		&b.Amount, &b.Currency, &b.Period, &startDate, &b.RolloverEnabled,
		&b.CreatedAt, &b.UpdatedAt,
	); err != nil {
		return nil, err
	}

	b.CategoryID = categoryID
	b.StartDate = startDate
	if icon.Valid {
		value := icon.String
		b.CategoryIcon = &value
	}
	if color.Valid {
		value := color.String
		b.CategoryColor = &value
	}
	b.Period = strings.ToUpper(strings.TrimSpace(b.Period))
	return b, nil
}
