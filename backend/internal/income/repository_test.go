package income

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

// ── mock infrastructure ──

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
			case *sql.NullTime:
				d.Valid = false
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
		case *sql.NullTime:
			d.Time = val.(time.Time)
			d.Valid = true
		default:
			return fmt.Errorf("unsupported scan type %T", d)
		}
	}
	return nil
}

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
			case *sql.NullTime:
				d.Valid = false
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
		case *sql.NullTime:
			d.Time = val.(time.Time)
			d.Valid = true
		default:
			return fmt.Errorf("unsupported scan type %T", d)
		}
	}
	return nil
}

type mockDB struct {
	execTag    pgconn.CommandTag
	execErr    error
	queryRows  [][]any
	queryErr   error
	beginTxErr error

	queryRowFn  func(ctx context.Context, sql string, args ...any) pgx.Row
	queryRowsFn func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (m *mockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return m.execTag, m.execErr
}
func (m *mockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryRowsFn != nil {
		return m.queryRowsFn(ctx, sql, args...)
	}
	return &mockRows{data: m.queryRows, err: m.queryErr}, nil
}
func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &mockRow{err: pgx.ErrNoRows}
}
func (m *mockDB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	if m.beginTxErr != nil {
		return nil, m.beginTxErr
	}
	return &mockTx{db: m}, nil
}

type mockTx struct {
	pgx.Tx
	db *mockDB
}

func (m *mockTx) Commit(ctx context.Context) error   { return nil }
func (m *mockTx) Rollback(ctx context.Context) error  { return nil }
func (m *mockTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return m.db.Exec(ctx, sql, args...)
}
func (m *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return m.db.QueryRow(ctx, sql, args...)
}

// ── repository tests ──

func TestRepositoryCreateIncome(t *testing.T) {
	now := time.Now()
	db := &mockDB{execTag: pgconn.NewCommandTag("UPDATE 1")}

	queryRowCount := 0
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		queryRowCount++
		if queryRowCount == 1 {
			return &mockRow{val: []any{500.0}} // wallet balance
		}
		return &mockRow{val: []any{now, now, nil}} // RETURNING
	}
	repo := NewRepository(db)

	inc := &Income{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		WalletID:   uuid.New(),
		SourceName: "Salary",
		Amount:     1000.0,
		Currency:   "USD",
		IncomeDate: now,
	}
	err := repo.CreateIncome(context.Background(), inc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepositoryCreateIncomeNil(t *testing.T) {
	db := &mockDB{}
	repo := NewRepository(db)
	err := repo.CreateIncome(context.Background(), nil)
	if err == nil {
		t.Errorf("expected error for nil income")
	}
}

func TestRepositoryCreateIncomeBeginTxError(t *testing.T) {
	db := &mockDB{beginTxErr: fmt.Errorf("begin tx error")}
	repo := NewRepository(db)
	err := repo.CreateIncome(context.Background(), &Income{ID: uuid.New()})
	if err == nil {
		t.Errorf("expected begin tx error")
	}
}

func TestRepositoryCreateIncomeWalletNotFound(t *testing.T) {
	db := &mockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &mockRow{err: pgx.ErrNoRows}
	}
	repo := NewRepository(db)
	err := repo.CreateIncome(context.Background(), &Income{ID: uuid.New(), WalletID: uuid.New()})
	if err == nil {
		t.Errorf("expected wallet not found error")
	}
}

func TestRepositoryGetIncomeByID(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uid := uuid.New()
	wid := uuid.New()
	cid := uuid.New().String()

	db := &mockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &mockRow{val: []any{id, uid, wid, cid, "Salary", 1000.0, "USD", now, "notes", false, now, now, nil}}
	}
	repo := NewRepository(db)

	inc, err := repo.GetIncomeByID(context.Background(), uid, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inc.ID != id || inc.Amount != 1000.0 {
		t.Errorf("unexpected income: %+v", inc)
	}
}

func TestRepositoryGetIncomeByIDNotFound(t *testing.T) {
	db := &mockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &mockRow{err: pgx.ErrNoRows}
	}
	repo := NewRepository(db)
	_, err := repo.GetIncomeByID(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Errorf("expected not found error")
	}
}

func TestRepositoryListIncomes(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uid := uuid.New()
	wid := uuid.New()
	cid := uuid.New().String()

	db := &mockDB{
		queryRows: [][]any{
			{id, uid, wid, cid, "Salary", 1000.0, "USD", now, "notes", false, now, now, nil},
		},
	}
	repo := NewRepository(db)

	list, pagination, err := repo.ListIncomes(context.Background(), uid, 10, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Errorf("unexpected list: %+v, pagination: %+v", list, pagination)
	}
}

func TestRepositoryListIncomesQueryError(t *testing.T) {
	db := &mockDB{}
	db.queryRowsFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
		return nil, fmt.Errorf("db error")
	}
	repo := NewRepository(db)
	_, _, err := repo.ListIncomes(context.Background(), uuid.New(), 10, nil)
	if err == nil {
		t.Errorf("expected query error")
	}
}

func TestRepositoryUpdateIncome(t *testing.T) {
	now := time.Now()
	uid := uuid.New()
	id := uuid.New()
	wid := uuid.New()

	db := &mockDB{execTag: pgconn.NewCommandTag("UPDATE 1")}
	queryRowCount := 0
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		queryRowCount++
		if queryRowCount == 1 {
			return &mockRow{val: []any{wid, 500.0}} // old wallet+amount
		}
		if queryRowCount == 2 {
			return &mockRow{val: []any{1000.0}} // wallet balance
		}
		return &mockRow{val: []any{now, now, nil}} // RETURNING
	}
	repo := NewRepository(db)

	inc := &Income{
		ID:         id,
		UserID:     uid,
		WalletID:   wid,
		SourceName: "Updated",
		Amount:     600.0,
		Currency:   "USD",
		IncomeDate: now,
	}
	err := repo.UpdateIncome(context.Background(), inc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepositoryUpdateIncomeNil(t *testing.T) {
	db := &mockDB{}
	repo := NewRepository(db)
	err := repo.UpdateIncome(context.Background(), nil)
	if err == nil {
		t.Errorf("expected error for nil income")
	}
}

func TestRepositoryUpdateIncomeBeginTxError(t *testing.T) {
	db := &mockDB{beginTxErr: fmt.Errorf("begin tx error")}
	repo := NewRepository(db)
	err := repo.UpdateIncome(context.Background(), &Income{ID: uuid.New()})
	if err == nil {
		t.Errorf("expected begin tx error")
	}
}

func TestRepositoryUpdateIncomeNotFound(t *testing.T) {
	db := &mockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &mockRow{err: pgx.ErrNoRows}
	}
	repo := NewRepository(db)
	err := repo.UpdateIncome(context.Background(), &Income{ID: uuid.New()})
	if err == nil {
		t.Errorf("expected not found error")
	}
}

func TestRepositoryUpdateIncomeWalletChange(t *testing.T) {
	now := time.Now()
	uid := uuid.New()
	id := uuid.New()
	wid1 := uuid.New()
	wid2 := uuid.New()

	db := &mockDB{execTag: pgconn.NewCommandTag("UPDATE 1")}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		if len(args) == 2 && args[1] == id {
			return &mockRow{val: []any{wid1, 500.0}} // old wallet+amount
		}
		if len(args) >= 1 {
			if wID, ok := args[0].(uuid.UUID); ok {
				if wID == wid1 {
					return &mockRow{val: []any{1000.0}}
				}
				if wID == wid2 {
					return &mockRow{val: []any{2000.0}}
				}
			}
		}
		return &mockRow{val: []any{now, now, nil}} // RETURNING
	}
	repo := NewRepository(db)

	inc := &Income{
		ID:         id,
		UserID:     uid,
		WalletID:   wid2, // changed wallet
		SourceName: "Updated",
		Amount:     600.0,
		Currency:   "USD",
		IncomeDate: now,
	}
	err := repo.UpdateIncome(context.Background(), inc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepositorySoftDeleteIncome(t *testing.T) {
	uid := uuid.New()
	id := uuid.New()
	wid := uuid.New()

	db := &mockDB{execTag: pgconn.NewCommandTag("UPDATE 1")}
	queryRowCount := 0
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		queryRowCount++
		if queryRowCount == 1 {
			return &mockRow{val: []any{wid, 500.0}} // wallet+amount
		}
		if queryRowCount == 2 {
			return &mockRow{val: []any{1000.0}} // wallet balance
		}
		return &mockRow{val: []any{id}} // RETURNING id
	}
	repo := NewRepository(db)

	err := repo.SoftDeleteIncome(context.Background(), uid, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepositorySoftDeleteIncomeBeginTxError(t *testing.T) {
	db := &mockDB{beginTxErr: fmt.Errorf("begin tx error")}
	repo := NewRepository(db)
	err := repo.SoftDeleteIncome(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Errorf("expected begin tx error")
	}
}

func TestRepositorySoftDeleteIncomeNotFound(t *testing.T) {
	db := &mockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &mockRow{err: pgx.ErrNoRows}
	}
	repo := NewRepository(db)
	err := repo.SoftDeleteIncome(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Errorf("expected not found error")
	}
}
