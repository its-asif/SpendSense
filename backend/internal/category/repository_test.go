package category

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ── mock infrastructure ──────────────────────────────────────────────────────

type cMockRow struct {
	val []any
	err error
}

func (m *cMockRow) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}
	for i, v := range m.val {
		if v == nil {
			switch d := dest[i].(type) {
			case *sql.NullString:
				d.Valid = false
			}
			continue
		}
		switch d := dest[i].(type) {
		case *uuid.UUID:
			*d = v.(uuid.UUID)
		case *string:
			*d = v.(string)
		case *bool:
			*d = v.(bool)
		case *time.Time:
			*d = v.(time.Time)
		case *sql.NullString:
			d.String = v.(string)
			d.Valid = true
		}
	}
	return nil
}

type cMockRows struct {
	pgx.Rows
	data [][]any
	idx  int
}

func (m *cMockRows) Next() bool { return m.idx < len(m.data) }
func (m *cMockRows) Close()     {}
func (m *cMockRows) Err() error { return nil }
func (m *cMockRows) Scan(dest ...any) error {
	row := m.data[m.idx]
	m.idx++
	for i, v := range row {
		if v == nil {
			switch d := dest[i].(type) {
			case *sql.NullString:
				d.Valid = false
			}
			continue
		}
		switch d := dest[i].(type) {
		case *uuid.UUID:
			*d = v.(uuid.UUID)
		case *string:
			*d = v.(string)
		case *bool:
			*d = v.(bool)
		case *time.Time:
			*d = v.(time.Time)
		case *sql.NullString:
			d.String = v.(string)
			d.Valid = true
		}
	}
	return nil
}

type cMockDB struct {
	queryRowFn  func(ctx context.Context, sql string, args ...any) pgx.Row
	queryRowsFn func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	execTag     pgconn.CommandTag
	execErr     error
}

func (m *cMockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return m.execTag, m.execErr
}
func (m *cMockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryRowsFn != nil {
		return m.queryRowsFn(ctx, sql, args...)
	}
	return &cMockRows{}, nil
}
func (m *cMockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &cMockRow{err: pgx.ErrNoRows}
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestRepositoryCreateCategory(t *testing.T) {
	now := time.Now()
	db := &cMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &cMockRow{val: []any{now}}
	}
	repo := NewRepository(db)
	uid := uuid.New()
	icon := "🍕"
	color := "#FF0000"
	c := &Category{ID: uuid.New(), Name: "Food", Kind: KindExpense, Icon: &icon, Color: &color}
	if err := repo.CreateCategory(context.Background(), uid, c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.UserID == nil || *c.UserID != uid {
		t.Errorf("expected UserID to be set after create")
	}
}

func TestRepositoryCreateCategoryScanError(t *testing.T) {
	db := &cMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &cMockRow{err: fmt.Errorf("db error")}
	}
	repo := NewRepository(db)
	err := repo.CreateCategory(context.Background(), uuid.New(), &Category{ID: uuid.New(), Name: "X"})
	if err == nil {
		t.Errorf("expected scan error")
	}
}

func TestRepositoryGetCategoryByID(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uid := uuid.New()
	db := &cMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &cMockRow{val: []any{id, uid.String(), "Food", nil, nil, "EXPENSE", false, now}}
	}
	repo := NewRepository(db)
	c, err := repo.GetCategoryByID(context.Background(), id, &uid, "EXPENSE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID != id || c.Name != "Food" {
		t.Errorf("unexpected category: %+v", c)
	}
}

func TestRepositoryGetCategoryByIDNotFound(t *testing.T) {
	db := &cMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &cMockRow{err: pgx.ErrNoRows}
	}
	repo := NewRepository(db)
	_, err := repo.GetCategoryByID(context.Background(), uuid.New(), nil, "")
	if err == nil {
		t.Errorf("expected not-found error")
	}
}

func TestRepositoryGetCategoryByIDNoUserIDNoKind(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	db := &cMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &cMockRow{val: []any{id, nil, "Food", nil, nil, "EXPENSE", true, now}}
	}
	repo := NewRepository(db)
	// No userID, no kind filter
	c, err := repo.GetCategoryByID(context.Background(), id, nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.IsDefault != true {
		t.Errorf("expected default category")
	}
}

func TestRepositoryListCategories(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uid := uuid.New()
	db := &cMockDB{}
	db.queryRowsFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
		return &cMockRows{data: [][]any{
			{id, uid.String(), "Food", nil, nil, "EXPENSE", false, now},
		}}, nil
	}
	repo := NewRepository(db)
	list, err := repo.ListCategories(context.Background(), uid, "EXPENSE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Errorf("unexpected list: %+v", list)
	}
}

func TestRepositoryListCategoriesQueryError(t *testing.T) {
	db := &cMockDB{}
	db.queryRowsFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
		return nil, fmt.Errorf("db error")
	}
	repo := NewRepository(db)
	_, err := repo.ListCategories(context.Background(), uuid.New(), "EXPENSE")
	if err == nil {
		t.Errorf("expected query error")
	}
}

func TestRepositoryUpdateCategory(t *testing.T) {
	now := time.Now()
	db := &cMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &cMockRow{val: []any{now, "EXPENSE"}}
	}
	repo := NewRepository(db)
	c := &Category{ID: uuid.New(), Name: "Updated"}
	if err := repo.UpdateCategory(context.Background(), uuid.New(), c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepositoryUpdateCategoryNotFound(t *testing.T) {
	db := &cMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &cMockRow{err: pgx.ErrNoRows}
	}
	repo := NewRepository(db)
	err := repo.UpdateCategory(context.Background(), uuid.New(), &Category{ID: uuid.New()})
	if err == nil {
		t.Errorf("expected not-found error")
	}
}

func TestRepositoryDeleteCategory(t *testing.T) {
	db := &cMockDB{execTag: pgconn.NewCommandTag("DELETE 1")}
	repo := NewRepository(db)
	if err := repo.DeleteCategory(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepositoryDeleteCategoryNotFound(t *testing.T) {
	db := &cMockDB{execTag: pgconn.NewCommandTag("DELETE 0")}
	repo := NewRepository(db)
	err := repo.DeleteCategory(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Errorf("expected not-found error")
	}
}

func TestRepositoryDeleteCategoryExecError(t *testing.T) {
	db := &cMockDB{execErr: fmt.Errorf("db error")}
	repo := NewRepository(db)
	err := repo.DeleteCategory(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Errorf("expected exec error")
	}
}
