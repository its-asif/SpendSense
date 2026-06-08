package budget

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

type mockRows struct {
	pgx.Rows
	data [][]any
	idx  int
	err  error
}

func (m *mockRows) Next() bool { return m.idx < len(m.data) }
func (m *mockRows) Close()     {}
func (m *mockRows) Err() error { return m.err }

func (m *mockRows) Scan(dest ...any) error {
	if m.idx >= len(m.data) {
		return fmt.Errorf("no more rows")
	}
	row := m.data[m.idx]
	m.idx++
	for i, val := range row {
		if val == nil {
			switch d := dest[i].(type) {
			case *sql.NullString:
				d.Valid = false
			}
			continue
		}
		switch d := dest[i].(type) {
		case *float64:
			*d = val.(float64)
		case *string:
			*d = val.(string)
		case *time.Time:
			*d = val.(time.Time)
		case *uuid.UUID:
			*d = val.(uuid.UUID)
		case *bool:
			*d = val.(bool)
		case *sql.NullString:
			d.String = val.(string)
			d.Valid = true
		default:
			return fmt.Errorf("unsupported scan type %T", d)
		}
	}
	return nil
}

type mockRow struct {
	val []any
	err error
}

func (m *mockRow) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}
	for i, val := range m.val {
		if val == nil {
			switch d := dest[i].(type) {
			case *sql.NullString:
				d.Valid = false
			}
			continue
		}
		switch d := dest[i].(type) {
		case *int:
			*d = val.(int)
		case *float64:
			*d = val.(float64)
		case *string:
			*d = val.(string)
		case *time.Time:
			*d = val.(time.Time)
		case *uuid.UUID:
			*d = val.(uuid.UUID)
		case *bool:
			*d = val.(bool)
		case *sql.NullString:
			d.String = val.(string)
			d.Valid = true
		default:
			return fmt.Errorf("unsupported scan type %T", d)
		}
	}
	return nil
}

type mockDB struct {
	execTag     pgconn.CommandTag
	execErr     error
	queryRows   [][]any
	queryErr    error
	queryRowVal []any
	queryRowErr error
}

func (m *mockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return m.execTag, m.execErr
}

func (m *mockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return &mockRows{data: m.queryRows, err: m.queryErr}, nil
}

func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return &mockRow{val: m.queryRowVal, err: m.queryRowErr}
}

func TestRepositoryCategoryAccessible(t *testing.T) {
	db := &mockDB{queryRowVal: []any{1}}
	repo := NewRepository(db)
	ok, err := repo.CategoryAccessible(context.Background(), uuid.New(), uuid.New())
	if err != nil || !ok {
		t.Errorf("expected accessible category")
	}

	db.queryRowErr = pgx.ErrNoRows
	ok, err = repo.CategoryAccessible(context.Background(), uuid.New(), uuid.New())
	if err != nil || ok {
		t.Errorf("expected not accessible category")
	}
}

func TestRepositoryHasMonthlyBudgetForCategory(t *testing.T) {
	db := &mockDB{queryRowVal: []any{1}}
	repo := NewRepository(db)
	ok, err := repo.HasMonthlyBudgetForCategory(context.Background(), uuid.New(), uuid.New(), nil)
	if err != nil || !ok {
		t.Errorf("expected budget exists")
	}
}

func TestRepositoryCreate(t *testing.T) {
	now := time.Now()
	db := &mockDB{queryRowVal: []any{now, now}}
	repo := NewRepository(db)
	b := &Budget{ID: uuid.New()}
	err := repo.Create(context.Background(), b)
	if err != nil {
		t.Errorf("unexpected error on Create: %v", err)
	}
}

func TestRepositoryGetByID(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uid := uuid.New()
	cid := uuid.New()
	db := &mockDB{
		queryRowVal: []any{id, uid, cid, "Category", "icon", "color", 100.0, "USD", "MONTHLY", now, true, now, now},
	}
	repo := NewRepository(db)
	b, err := repo.GetByID(context.Background(), uid, id)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if b.ID != id || b.Amount != 100.0 {
		t.Errorf("unexpected budget retrieved: %+v", b)
	}
}

func TestRepositoryList(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uid := uuid.New()
	cid := uuid.New()
	db := &mockDB{
		queryRows: [][]any{
			{id, uid, cid, "Category", "icon", "color", 100.0, "USD", "MONTHLY", now, true, now, now},
		},
	}
	repo := NewRepository(db)
	list, err := repo.List(context.Background(), uid, "MONTHLY")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Errorf("unexpected list: %+v", list)
	}
}

func TestRepositoryUpdate(t *testing.T) {
	now := time.Now()
	db := &mockDB{queryRowVal: []any{now, now}}
	repo := NewRepository(db)
	b := &Budget{ID: uuid.New(), Amount: 150}
	err := repo.Update(context.Background(), b)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRepositoryDelete(t *testing.T) {
	db := &mockDB{execTag: pgconn.NewCommandTag("DELETE 1")}
	repo := NewRepository(db)
	err := repo.Delete(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	db.execTag = pgconn.NewCommandTag("DELETE 0")
	err = repo.Delete(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Errorf("expected error for zero rows affected")
	}
}
