package expense

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
	execTag     pgconn.CommandTag
	execErr     error
	queryRows   [][]any
	queryErr    error
	queryRowVal []any
	queryRowErr error
	beginTxErr  error

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
	return &mockRow{val: m.queryRowVal, err: m.queryRowErr}
}

func (m *mockDB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return &mockTx{db: m}, m.beginTxErr
}

type mockTx struct {
	pgx.Tx
	db *mockDB
}

func (m *mockTx) Commit(ctx context.Context) error   { return nil }
func (m *mockTx) Rollback(ctx context.Context) error { return nil }
func (m *mockTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return m.db.Exec(ctx, sql, args...)
}
func (m *mockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return m.db.QueryRow(ctx, sql, args...)
}

func TestRepositoryCreateExpense(t *testing.T) {
	db := &mockDB{
		execTag: pgconn.NewCommandTag("UPDATE 1"),
	}
	repo := NewRepository(db)

	now := time.Now()

	e := &Expense{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		WalletID:   uuid.New(),
		CategoryID: uuid.New(),
		Amount:     100.0,
		Currency:   "USD",
		Date:       now,
	}

	queryRowCount := 0
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		queryRowCount++
		if queryRowCount == 1 {
			return &mockRow{val: []any{500.0}}
		}
		return &mockRow{val: []any{now, now, nil}}
	}

	err := repo.CreateExpense(context.Background(), e)
	if err != nil {
		t.Fatalf("unexpected error creating expense: %v", err)
	}

	// Test insufficient balance
	queryRowCount = 0
	e.Amount = 600.0
	err = repo.CreateExpense(context.Background(), e)
	if err == nil {
		t.Errorf("expected insufficient balance error")
	}
}

func TestRepositoryGetExpenseByID(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uid := uuid.New()
	wid := uuid.New()
	cid := uuid.New()

	db := &mockDB{
		queryRowVal: []any{id, uid, wid, 150.0, "USD", 1.0, cid, "merchant", now, "notes", false, "rule", false, now, now, nil},
	}
	repo := NewRepository(db)

	exp, err := repo.GetExpenseByID(context.Background(), uid, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.ID != id || exp.Amount != 150.0 {
		t.Errorf("unexpected expense: %+v", exp)
	}
}

func TestRepositoryListExpenses(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uid := uuid.New()
	wid := uuid.New()
	cid := uuid.New()

	db := &mockDB{
		queryRows: [][]any{
			{id, uid, wid, 150.0, "USD", 1.0, cid, "merchant", now, "notes", false, "rule", false, now, now, nil},
		},
	}
	repo := NewRepository(db)

	list, pagination, err := repo.ListExpenses(context.Background(), uid, 10, nil, &now, &now, &cid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Errorf("unexpected list or pagination: %+v, %+v", list, pagination)
	}
}

func TestRepositoryUpdateExpense(t *testing.T) {
	db := &mockDB{
		execTag: pgconn.NewCommandTag("UPDATE 1"),
	}
	now := time.Now()
	uid := uuid.New()
	id := uuid.New()
	wid := uuid.New()
	cid := uuid.New()

	e := &Expense{
		ID:         id,
		UserID:     uid,
		WalletID:   wid,
		CategoryID: cid,
		Amount:     200.0,
		Currency:   "USD",
		Date:       now,
	}

	queryRowCount := 0
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		queryRowCount++
		if queryRowCount == 1 {
			// Select wallet_id, amount from old expense
			return &mockRow{val: []any{wid, 100.0}}
		}
		if queryRowCount == 2 {
			// Lock balance for wallet
			return &mockRow{val: []any{500.0}}
		}
		// Insert/Update returning
		return &mockRow{val: []any{now, now, nil}}
	}
	repo := NewRepository(db)

	err := repo.UpdateExpense(context.Background(), e)
	if err != nil {
		t.Fatalf("unexpected error updating expense: %v", err)
	}
}

func TestRepositorySoftDeleteExpense(t *testing.T) {
	db := &mockDB{
		execTag: pgconn.NewCommandTag("UPDATE 1"),
	}
	uid := uuid.New()
	id := uuid.New()
	wid := uuid.New()

	queryRowCount := 0
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		queryRowCount++
		if queryRowCount == 1 {
			// Select wallet_id, amount from old expense
			return &mockRow{val: []any{wid, 100.0}}
		}
		if queryRowCount == 2 {
			// Lock balance for wallet
			return &mockRow{val: []any{500.0}}
		}
		// Soft delete returning ID
		return &mockRow{val: []any{id}}
	}
	repo := NewRepository(db)

	err := repo.SoftDeleteExpense(context.Background(), uid, id)
	if err != nil {
		t.Fatalf("unexpected error soft deleting expense: %v", err)
	}
}

func TestRepositoryAdditionalPaths(t *testing.T) {
	db := &mockDB{
		execTag: pgconn.NewCommandTag("UPDATE 1"),
	}
	repo := NewRepository(db)

	// 1. Nil parameter checks
	t.Run("NilInputs", func(t *testing.T) {
		if err := repo.CreateExpense(context.Background(), nil); err == nil {
			t.Errorf("expected error when creating nil expense")
		}
		if err := repo.UpdateExpense(context.Background(), nil); err == nil {
			t.Errorf("expected error when updating nil expense")
		}
	})

	// 2. lockWalletBalances validation
	t.Run("LockWalletValidation", func(t *testing.T) {
		e := &Expense{
			ID:       uuid.New(),
			UserID:   uuid.New(),
			WalletID: uuid.Nil, // Nil wallet
			Amount:   10.0,
		}
		db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{val: []any{500.0}}
		}
		err := repo.CreateExpense(context.Background(), e)
		if err == nil {
			t.Errorf("expected error for nil wallet ID")
		}
	})

	// 3. Update expense with wallet change
	t.Run("UpdateExpenseWalletChange", func(t *testing.T) {
		uid := uuid.New()
		id := uuid.New()
		wid1 := uuid.New()
		wid2 := uuid.New()
		now := time.Now()

		e := &Expense{
			ID:         id,
			UserID:     uid,
			WalletID:   wid2,
			CategoryID: uuid.New(),
			Amount:     50.0,
			Date:       now,
		}

		db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			if len(args) == 2 && args[1] == id {
				// Querying old expense
				return &mockRow{val: []any{wid1, 100.0}}
			}
			if len(args) >= 1 {
				if wID, ok := args[0].(uuid.UUID); ok {
					if wID == wid1 {
						return &mockRow{val: []any{500.0}}
					}
					if wID == wid2 {
						return &mockRow{val: []any{200.0}}
					}
				}
			}
			return &mockRow{val: []any{now, now, nil}}
		}

		err := repo.UpdateExpense(context.Background(), e)
		if err != nil {
			t.Fatalf("unexpected error updating wallet change: %v", err)
		}
	})

	// 4. Update expense wallet change insufficient balance
	t.Run("UpdateExpenseWalletChangeInsufficient", func(t *testing.T) {
		uid := uuid.New()
		id := uuid.New()
		wid1 := uuid.New()
		wid2 := uuid.New()
		now := time.Now()

		e := &Expense{
			ID:         id,
			UserID:     uid,
			WalletID:   wid2,
			CategoryID: uuid.New(),
			Amount:     300.0, // Exceeds wallet 2 balance
			Date:       now,
		}

		db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			if len(args) == 2 && args[1] == id {
				// Querying old expense
				return &mockRow{val: []any{wid1, 100.0}}
			}
			if len(args) >= 1 {
				if wID, ok := args[0].(uuid.UUID); ok {
					if wID == wid1 {
						return &mockRow{val: []any{500.0}}
					}
					if wID == wid2 {
						return &mockRow{val: []any{100.0}}
					}
				}
			}
			return &mockRow{val: []any{now, now, nil}}
		}

		err := repo.UpdateExpense(context.Background(), e)
		if err == nil {
			t.Errorf("expected insufficient balance error on wallet change")
		}
	})

	// 5. Query execution errors and DB errors
	t.Run("QueryAndDBErrors", func(t *testing.T) {
		uid := uuid.New()
		id := uuid.New()

		// GetExpenseByID not found
		db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{err: pgx.ErrNoRows}
		}
		_, err := repo.GetExpenseByID(context.Background(), uid, id)
		if err == nil {
			t.Errorf("expected error for non-existent expense ID")
		}

		// ListExpenses query execution error
		db.queryRowsFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
			return nil, fmt.Errorf("db error")
		}
		db.queryRowFn = nil
		_, _, err = repo.ListExpenses(context.Background(), uid, 10, nil, nil, nil, nil)
		if err == nil {
			t.Errorf("expected query execution error")
		}

		// Transaction failure on Create
		dbErr := &mockDB{beginTxErr: fmt.Errorf("begin tx error")}
		repoErr := NewRepository(dbErr)
		err = repoErr.CreateExpense(context.Background(), &Expense{ID: uuid.New()})
		if err == nil {
			t.Errorf("expected transaction begin error on Create")
		}

		// Transaction failure on Update
		err = repoErr.UpdateExpense(context.Background(), &Expense{ID: uuid.New()})
		if err == nil {
			t.Errorf("expected transaction begin error on Update")
		}

		// Transaction failure on SoftDelete
		err = repoErr.SoftDeleteExpense(context.Background(), uuid.New(), uuid.New())
		if err == nil {
			t.Errorf("expected transaction begin error on SoftDelete")
		}

		// Wallet not found during Create
		dbWalletErr := &mockDB{}
		dbWalletErr.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
			return &mockRow{err: pgx.ErrNoRows}
		}
		repoWalletErr := NewRepository(dbWalletErr)
		err = repoWalletErr.CreateExpense(context.Background(), &Expense{ID: uuid.New(), WalletID: uuid.New()})
		if err == nil {
			t.Errorf("expected wallet not found error on Create")
		}

		// Expense not found during Update
		err = repoWalletErr.UpdateExpense(context.Background(), &Expense{ID: uuid.New()})
		if err == nil {
			t.Errorf("expected expense not found error on Update")
		}

		// Expense not found during SoftDelete
		err = repoWalletErr.SoftDeleteExpense(context.Background(), uuid.New(), uuid.New())
		if err == nil {
			t.Errorf("expected expense not found error on SoftDelete")
		}
	})
}
