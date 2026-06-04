package infra

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	pool *pgxpool.Pool
}

func NewDatabase(connStr string) (*Database, error) {
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	// Ping db
	if err = pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed pinging database: %w", err)
	}

	return &Database{pool: pool}, nil
}

func (db *Database) Close() {
	db.pool.Close()
}

func (db *Database) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

func (db *Database) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

func (db *Database) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

// BeginTx starts a new transaction. Caller should call Commit or Rollback on the returned tx.
func (db *Database) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

// AutoMigrateRecurringPayments checks and creates the recurring_payments table if it doesn't exist.
func (db *Database) AutoMigrateRecurringPayments(ctx context.Context) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS recurring_payments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
			category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			amount NUMERIC(18,2) NOT NULL CHECK (amount > 0),
			currency VARCHAR(3) NOT NULL DEFAULT 'USD',
			interval VARCHAR(20) NOT NULL DEFAULT 'monthly',
			start_date DATE NOT NULL,
			deadline DATE NOT NULL,
			alert_rule VARCHAR(10) NOT NULL DEFAULT '1d',
			end_date DATE,
			status VARCHAR(20) NOT NULL DEFAULT 'unpaid',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}
	_, _ = db.Exec(ctx, "CREATE INDEX IF NOT EXISTS idx_recurring_payments_user_id ON recurring_payments(user_id)")
	_, _ = db.Exec(ctx, "CREATE INDEX IF NOT EXISTS idx_recurring_payments_wallet_id ON recurring_payments(wallet_id)")
	_, _ = db.Exec(ctx, "CREATE INDEX IF NOT EXISTS idx_recurring_payments_status ON recurring_payments(status)")
	return nil
}
