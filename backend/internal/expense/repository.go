package expense

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/infra"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Store interface {
	CreateExpense(ctx context.Context, expense *Expense) error
	GetExpenseByID(ctx context.Context, userID, expenseID uuid.UUID) (*Expense, error)
	ListExpenses(ctx context.Context, userID uuid.UUID, limit int, pagination *Pagination, from, to *time.Time, categoryID *uuid.UUID) ([]*Expense, *Pagination, error)
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

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	balances, err := lockWalletBalances(ctx, tx, expense.UserID, expense.WalletID)
	if err != nil {
		return err
	}

	if balances[expense.WalletID] < expense.Amount {
		return domain.NewDomainError(domain.ErrInvalidAmount, "insufficient wallet balance for expense", 400)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE wallets
		SET current_balance = current_balance - $1,
			updated_at = NOW()
		WHERE id = $2 AND user_id = $3
	`, expense.Amount, expense.WalletID, expense.UserID); err != nil {
		return err
	}

	row := tx.QueryRow(ctx, `
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

	if err := tx.Commit(ctx); err != nil {
		return err
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

func (r *Repository) ListExpenses(ctx context.Context, userID uuid.UUID, limit int, pagination *Pagination, from, to *time.Time, categoryID *uuid.UUID) ([]*Expense, *Pagination, error) {
	query := `
		SELECT id, user_id, wallet_id, amount, currency, fx_rate_to_base, category_id,
			merchant, date, notes, is_recurring, recurring_rule, is_deleted,
			created_at, updated_at, deleted_at
		FROM expenses
		WHERE user_id = $1 AND is_deleted = FALSE
	`

	args := []any{userID}
	argPos := 2
	if from != nil {
		query += fmt.Sprintf(" AND date >= $%d", argPos)
		args = append(args, *from)
		argPos++
	}
	if to != nil {
		query += fmt.Sprintf(" AND date <= $%d", argPos)
		args = append(args, *to)
		argPos++
	}
	if categoryID != nil && *categoryID != uuid.Nil {
		query += fmt.Sprintf(" AND category_id = $%d", argPos)
		args = append(args, *categoryID)
		argPos++
	}
	if pagination != nil {
		query += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argPos, argPos+1)
		args = append(args, pagination.CreatedAt, pagination.ID)
		argPos += 2
	}

	args = append(args, limit+1)
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", argPos)

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

	var nextPagination *Pagination
	if len(expenses) > limit {
		last := expenses[limit-1]
		nextPagination = &Pagination{CreatedAt: last.CreatedAt, ID: last.ID}
		expenses = expenses[:limit]
	}

	return expenses, nextPagination, nil
}

func (r *Repository) UpdateExpense(ctx context.Context, expense *Expense) error {
	if expense == nil {
		return domain.NewDomainError(domain.ErrInternal, "expense is required", 400)
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var oldWalletID uuid.UUID
	var oldAmount float64
	row := tx.QueryRow(ctx, `
		SELECT wallet_id, amount
		FROM expenses
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
		FOR UPDATE
	`, expense.UserID, expense.ID)
	if err := row.Scan(&oldWalletID, &oldAmount); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "expense not found", 404)
		}
		return err
	}

	if oldWalletID == expense.WalletID {
		balances, err := lockWalletBalances(ctx, tx, expense.UserID, oldWalletID)
		if err != nil {
			return err
		}

		newBalance := balances[oldWalletID] + oldAmount - expense.Amount
		if newBalance < 0 {
			return domain.NewDomainError(domain.ErrInvalidAmount, "insufficient wallet balance for expense update", 400)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE wallets
			SET current_balance = $1,
				updated_at = NOW()
			WHERE id = $2 AND user_id = $3
		`, round2(newBalance), oldWalletID, expense.UserID); err != nil {
			return err
		}
	} else {
		balances, err := lockWalletBalances(ctx, tx, expense.UserID, oldWalletID, expense.WalletID)
		if err != nil {
			return err
		}

		oldWalletBalance := balances[oldWalletID] + oldAmount
		newWalletBalance := balances[expense.WalletID] - expense.Amount
		if newWalletBalance < 0 {
			return domain.NewDomainError(domain.ErrInvalidAmount, "insufficient wallet balance for expense update", 400)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE wallets
			SET current_balance = $1,
				updated_at = NOW()
			WHERE id = $2 AND user_id = $3
		`, round2(oldWalletBalance), oldWalletID, expense.UserID); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			UPDATE wallets
			SET current_balance = $1,
				updated_at = NOW()
			WHERE id = $2 AND user_id = $3
		`, round2(newWalletBalance), expense.WalletID, expense.UserID); err != nil {
			return err
		}
	}

	row = tx.QueryRow(ctx, `
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

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (r *Repository) SoftDeleteExpense(ctx context.Context, userID, expenseID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var walletID uuid.UUID
	var amount float64
	row := tx.QueryRow(ctx, `
		SELECT wallet_id, amount
		FROM expenses
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
		FOR UPDATE
	`, userID, expenseID)
	if err := row.Scan(&walletID, &amount); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "expense not found", 404)
		}
		return err
	}

	balances, err := lockWalletBalances(ctx, tx, userID, walletID)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE wallets
		SET current_balance = $1,
			updated_at = NOW()
		WHERE id = $2 AND user_id = $3
	`, round2(balances[walletID]+amount), walletID, userID); err != nil {
		return err
	}

	row = tx.QueryRow(ctx, `
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

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func lockWalletBalances(ctx context.Context, tx pgx.Tx, userID uuid.UUID, walletIDs ...uuid.UUID) (map[uuid.UUID]float64, error) {
	unique := make(map[uuid.UUID]struct{}, len(walletIDs))
	ordered := make([]uuid.UUID, 0, len(walletIDs))
	for _, walletID := range walletIDs {
		if walletID == uuid.Nil {
			return nil, domain.NewDomainError(domain.ErrInvalidWallet, "wallet_id is required", 400)
		}
		if _, ok := unique[walletID]; ok {
			continue
		}
		unique[walletID] = struct{}{}
		ordered = append(ordered, walletID)
	}

	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].String() < ordered[j].String()
	})

	balances := make(map[uuid.UUID]float64, len(ordered))
	for _, walletID := range ordered {
		var balance float64
		row := tx.QueryRow(ctx, `
			SELECT current_balance
			FROM wallets
			WHERE id = $1 AND user_id = $2
			FOR UPDATE
		`, walletID, userID)
		if err := row.Scan(&balance); err != nil {
			if err == pgx.ErrNoRows {
				return nil, domain.NewDomainError(domain.ErrNotFound, "wallet not found", 404)
			}
			return nil, err
		}
		balances[walletID] = round2(balance)
	}

	return balances, nil
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
