package category

import (
	"context"
	"database/sql"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/infra"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db *infra.Database }

func NewRepository(db *infra.Database) *Repository { return &Repository{db: db} }

func (r *Repository) CreateCategory(ctx context.Context, userID uuid.UUID, c *Category) error {
	row := r.db.QueryRow(ctx, `
        INSERT INTO categories (id, user_id, name, icon, color, is_default)
        VALUES ($1,$2,$3,$4,$5,FALSE)
        RETURNING created_at
    `, c.ID, userID, c.Name, c.Icon, c.Color)
	if err := row.Scan(&c.CreatedAt); err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetCategoryByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*Category, error) {
	row := r.db.QueryRow(ctx, `
        SELECT id, user_id, name, icon, color, is_default, created_at FROM categories WHERE id=$1
    `, id)
	c := &Category{}
	var user sql.NullString
	var icon sql.NullString
	var color sql.NullString
	if err := row.Scan(&c.ID, &user, &c.Name, &icon, &color, &c.IsDefault, &c.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.NewDomainError(domain.ErrNotFound, "category not found", 404)
		}
		return nil, err
	}
	if user.Valid {
		uid, _ := uuid.Parse(user.String)
		c.UserID = &uid
	}
	if icon.Valid {
		v := icon.String
		c.Icon = &v
	}
	if color.Valid {
		v := color.String
		c.Color = &v
	}
	return c, nil
}

func (r *Repository) ListCategories(ctx context.Context, userID uuid.UUID) ([]*Category, error) {
	rows, err := r.db.Query(ctx, `
        SELECT id, user_id, name, icon, color, is_default, created_at FROM categories WHERE user_id=$1 OR is_default = TRUE ORDER BY is_default DESC, name
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []*Category{}
	for rows.Next() {
		c := &Category{}
		var user sql.NullString
		var icon sql.NullString
		var color sql.NullString
		if err := rows.Scan(&c.ID, &user, &c.Name, &icon, &color, &c.IsDefault, &c.CreatedAt); err != nil {
			return nil, err
		}
		if user.Valid {
			uid, _ := uuid.Parse(user.String)
			c.UserID = &uid
		}
		if icon.Valid {
			v := icon.String
			c.Icon = &v
		}
		if color.Valid {
			v := color.String
			c.Color = &v
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *Repository) UpdateCategory(ctx context.Context, userID uuid.UUID, c *Category) error {
	row := r.db.QueryRow(ctx, `
        UPDATE categories SET name=$3, icon=$4, color=$5 WHERE id=$1 AND user_id=$2 RETURNING created_at
    `, c.ID, userID, c.Name, c.Icon, c.Color)
	if err := row.Scan(&c.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "category not found", 404)
		}
		return err
	}
	return nil
}

func (r *Repository) DeleteCategory(ctx context.Context, userID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM categories WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrNotFound, "category not found", 404)
	}
	return nil
}
