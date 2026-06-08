package wallet

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

type wMockRow struct {
	val []any
	err error
}

func (m *wMockRow) Scan(dest ...any) error {
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
		case *float64:
			*d = v.(float64)
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

type wMockRows struct {
	pgx.Rows
	data [][]any
	idx  int
}

func (m *wMockRows) Next() bool { return m.idx < len(m.data) }
func (m *wMockRows) Close()     {}
func (m *wMockRows) Err() error { return nil }
func (m *wMockRows) Scan(dest ...any) error {
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
		case *float64:
			*d = v.(float64)
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

type wMockDB struct {
	queryRowFn  func(ctx context.Context, sql string, args ...any) pgx.Row
	queryRowsFn func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	execTag     pgconn.CommandTag
	execErr     error
	beginTxErr  error
}

func (m *wMockDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return m.execTag, m.execErr
}
func (m *wMockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryRowsFn != nil {
		return m.queryRowsFn(ctx, sql, args...)
	}
	return &wMockRows{}, nil
}
func (m *wMockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &wMockRow{err: pgx.ErrNoRows}
}
func (m *wMockDB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	if m.beginTxErr != nil {
		return nil, m.beginTxErr
	}
	return &wMockTx{db: m}, nil
}

type wMockTx struct {
	pgx.Tx
	db         *wMockDB
	committed  bool
	rolledBack bool
}

func (t *wMockTx) Commit(ctx context.Context) error   { t.committed = true; return nil }
func (t *wMockTx) Rollback(ctx context.Context) error  { t.rolledBack = true; return nil }
func (t *wMockTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return t.db.Exec(ctx, sql, args...)
}
func (t *wMockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.db.QueryRow(ctx, sql, args...)
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestRepositoryCreateWallet(t *testing.T) {
	now := time.Now()
	db := &wMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &wMockRow{val: []any{now, now}}
	}
	repo := NewRepository(db)
	w := &Wallet{ID: uuid.New(), UserID: uuid.New(), Name: "Test"}
	if err := repo.CreateWallet(context.Background(), w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepositoryCreateWalletNil(t *testing.T) {
	repo := NewRepository(&wMockDB{})
	if err := repo.CreateWallet(context.Background(), nil); err == nil {
		t.Errorf("expected error for nil wallet")
	}
}

func TestRepositoryGetWalletByID(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uid := uuid.New()
	db := &wMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &wMockRow{val: []any{id, uid, "Cash", "CASH", nil, nil, nil, "USD", 100.0, 100.0, true, now, now}}
	}
	repo := NewRepository(db)
	w, err := repo.GetWalletByID(context.Background(), uid, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.ID != id || w.Currency != "USD" {
		t.Errorf("unexpected wallet: %+v", w)
	}
}

func TestRepositoryGetWalletByIDNotFound(t *testing.T) {
	db := &wMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &wMockRow{err: pgx.ErrNoRows}
	}
	repo := NewRepository(db)
	_, err := repo.GetWalletByID(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Errorf("expected not-found error")
	}
}

func TestRepositoryListWallets(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	uid := uuid.New()
	db := &wMockDB{}
	db.queryRowsFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
		return &wMockRows{data: [][]any{
			{id, uid, "Cash", "CASH", nil, nil, nil, "USD", 100.0, 100.0, true, now, now},
		}}, nil
	}
	repo := NewRepository(db)
	list, err := repo.ListWallets(context.Background(), uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].ID != id {
		t.Errorf("unexpected list: %+v", list)
	}
}

func TestRepositoryListWalletsQueryError(t *testing.T) {
	db := &wMockDB{}
	db.queryRowsFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
		return nil, fmt.Errorf("db error")
	}
	repo := NewRepository(db)
	_, err := repo.ListWallets(context.Background(), uuid.New())
	if err == nil {
		t.Errorf("expected query error")
	}
}

func TestRepositoryUpdateWallet(t *testing.T) {
	now := time.Now()
	db := &wMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &wMockRow{val: []any{now, now}}
	}
	repo := NewRepository(db)
	w := &Wallet{ID: uuid.New(), UserID: uuid.New(), Name: "Updated", Currency: "USD"}
	if err := repo.UpdateWallet(context.Background(), w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepositoryUpdateWalletNotFound(t *testing.T) {
	db := &wMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &wMockRow{err: pgx.ErrNoRows}
	}
	repo := NewRepository(db)
	err := repo.UpdateWallet(context.Background(), &Wallet{ID: uuid.New()})
	if err == nil {
		t.Errorf("expected not-found error")
	}
}

func TestRepositoryDeleteWallet(t *testing.T) {
	db := &wMockDB{execTag: pgconn.NewCommandTag("DELETE 1")}
	repo := NewRepository(db)
	if err := repo.DeleteWallet(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepositoryDeleteWalletNotFound(t *testing.T) {
	db := &wMockDB{execTag: pgconn.NewCommandTag("DELETE 0")}
	repo := NewRepository(db)
	err := repo.DeleteWallet(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Errorf("expected not-found error")
	}
}

func TestRepositoryDeleteWalletExecError(t *testing.T) {
	db := &wMockDB{execErr: fmt.Errorf("db error")}
	repo := NewRepository(db)
	err := repo.DeleteWallet(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Errorf("expected exec error")
	}
}

func TestRepositoryCreateTransfer(t *testing.T) {
	now := time.Now()
	from := uuid.New()
	to := uuid.New()
	db := &wMockDB{execTag: pgconn.NewCommandTag("UPDATE 1")}
	callCount := 0
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		callCount++
		if callCount == 1 {
			return &wMockRow{val: []any{1500.0}} // from balance
		}
		if callCount == 2 {
			return &wMockRow{val: []any{500.0}} // to balance
		}
		return &wMockRow{val: []any{now}} // RETURNING created_at
	}
	repo := NewRepository(db)
	tr := &Transfer{
		ID: uuid.New(), UserID: uuid.New(),
		FromWalletID: from, ToWalletID: to,
		Amount: 100, FeeAmount: 5, ConvertedAmount: 100,
		Currency: "USD", TransferDate: now,
	}
	if err := repo.CreateTransfer(context.Background(), tr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepositoryCreateTransferNil(t *testing.T) {
	repo := NewRepository(&wMockDB{})
	if err := repo.CreateTransfer(context.Background(), nil); err == nil {
		t.Errorf("expected error for nil transfer")
	}
}

func TestRepositoryCreateTransferBeginTxError(t *testing.T) {
	db := &wMockDB{beginTxErr: fmt.Errorf("begin tx error")}
	repo := NewRepository(db)
	err := repo.CreateTransfer(context.Background(), &Transfer{ID: uuid.New()})
	if err == nil {
		t.Errorf("expected begin tx error")
	}
}

func TestRepositoryCreateTransferFromWalletNotFound(t *testing.T) {
	db := &wMockDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		return &wMockRow{err: pgx.ErrNoRows}
	}
	repo := NewRepository(db)
	err := repo.CreateTransfer(context.Background(), &Transfer{
		ID: uuid.New(), FromWalletID: uuid.New(), ToWalletID: uuid.New(),
	})
	if err == nil {
		t.Errorf("expected from-wallet not found error")
	}
}

func TestRepositoryCreateTransferInsufficientFunds(t *testing.T) {
	from := uuid.New()
	to := uuid.New()
	db := &wMockDB{execTag: pgconn.NewCommandTag("UPDATE 1")}
	callCount := 0
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		callCount++
		if callCount == 1 {
			return &wMockRow{val: []any{10.0}} // insufficient balance (amount=100 + fee=5)
		}
		return &wMockRow{val: []any{500.0}}
	}
	repo := NewRepository(db)
	err := repo.CreateTransfer(context.Background(), &Transfer{
		ID: uuid.New(), FromWalletID: from, ToWalletID: to,
		Amount: 100, FeeAmount: 5,
	})
	if err == nil {
		t.Errorf("expected insufficient funds error")
	}
}
