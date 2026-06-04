package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/infra"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db *infra.Database }

func NewRepository(db *infra.Database) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, n *Notification) error {
	row := r.db.QueryRow(ctx, `
		INSERT INTO notifications (id, user_id, type, title, body, metadata, is_read)
		VALUES ($1, $2, $3, $4, $5, $6, FALSE)
		RETURNING created_at
	`, n.ID, n.UserID, n.Type, n.Title, n.Body, n.Metadata)
	return row.Scan(&n.CreatedAt)
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID, limit int, unreadOnly bool) ([]*Notification, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT id, user_id, type, title, body, metadata, is_read, created_at, dismissed_at
		FROM notifications
		WHERE user_id = $1 AND dismissed_at IS NULL
	`
	if unreadOnly {
		query += ` AND is_read = FALSE`
	}
	query += ` ORDER BY created_at DESC LIMIT $2`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]*Notification, 0)
	for rows.Next() {
		n, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

func (r *Repository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	row := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE user_id = $1 AND is_read = FALSE AND dismissed_at IS NULL
	`, userID)
	var count int
	return count, row.Scan(&count)
}

func (r *Repository) MarkRead(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE
		WHERE id = $1 AND user_id = $2 AND dismissed_at IS NULL
	`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrNotFound, "notification not found", 404)
	}
	return nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE
		WHERE user_id = $1 AND dismissed_at IS NULL AND is_read = FALSE
	`, userID)
	return err
}

func (r *Repository) Dismiss(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET dismissed_at = CURRENT_TIMESTAMP, is_read = TRUE
		WHERE id = $1 AND user_id = $2 AND dismissed_at IS NULL
	`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrNotFound, "notification not found", 404)
	}
	return nil
}

func (r *Repository) BudgetAlertAlreadySent(ctx context.Context, budgetID uuid.UUID, threshold int, yearMonth string) (bool, error) {
	row := r.db.QueryRow(ctx, `
		SELECT 1 FROM budget_alerts_sent
		WHERE budget_id = $1 AND threshold_percent = $2 AND year_month = $3
		LIMIT 1
	`, budgetID, threshold, yearMonth)
	var marker int
	if err := row.Scan(&marker); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) RecordBudgetAlertSent(ctx context.Context, budgetID uuid.UUID, threshold int, yearMonth string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO budget_alerts_sent (id, budget_id, threshold_percent, year_month)
		VALUES (gen_random_uuid(), $1, $2, $3)
		ON CONFLICT (budget_id, threshold_percent, year_month) DO NOTHING
	`, budgetID, threshold, yearMonth)
	return err
}

func (r *Repository) ClearBudgetAlertSent(ctx context.Context, budgetID uuid.UUID, threshold int, yearMonth string) error {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete from budget_alerts_sent tracker
	_, err = tx.Exec(ctx, `
		DELETE FROM budget_alerts_sent
		WHERE budget_id = $1 AND threshold_percent = $2 AND year_month = $3
	`, budgetID, threshold, yearMonth)
	if err != nil {
		return err
	}

	// Delete stale notification
	_, err = tx.Exec(ctx, `
		DELETE FROM notifications
		WHERE type = 'budget_alert'
		  AND metadata->>'budget_id' = $1
		  AND metadata->>'threshold_percent' = $2
		  AND metadata->>'year_month' = $3
	`, budgetID.String(), strconv.Itoa(threshold), yearMonth)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}


func (r *Repository) scan(rows pgx.Rows) (*Notification, error) {
	n := &Notification{}
	var metadata []byte
	var dismissed sql.NullTime
	if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &metadata, &n.IsRead, &n.CreatedAt, &dismissed); err != nil {
		return nil, err
	}
	if len(metadata) > 0 {
		n.Metadata = json.RawMessage(metadata)
	}
	if dismissed.Valid {
		n.DismissedAt = &dismissed.Time
	}
	return n, nil
}
