package income

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"spendsense-backend/internal/domain"
	"spendsense-backend/internal/infra"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Store interface {
	CreateIncome(ctx context.Context, income *Income) error
	GetIncomeByID(ctx context.Context, userID, incomeID uuid.UUID) (*Income, error)
	ListIncomes(ctx context.Context, userID uuid.UUID, limit int, pagination *Pagination) ([]*Income, *Pagination, error)
	UpdateIncome(ctx context.Context, income *Income) error
	SoftDeleteIncome(ctx context.Context, userID, incomeID uuid.UUID) error
}

type Repository struct {
	db *infra.Database
}

func NewRepository(db *infra.Database) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateIncome(ctx context.Context, income *Income) error {
	if income == nil {
		return domain.NewDomainError(domain.ErrInternal, "income is required", 400)
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := lockWalletBalances(ctx, tx, income.UserID, income.WalletID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE wallets
		SET current_balance = current_balance + $1,
			updated_at = NOW()
		WHERE id = $2 AND user_id = $3
	`, income.Amount, income.WalletID, income.UserID); err != nil {
		return err
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO incomes (
			id, user_id, wallet_id, category_id, source_name, amount, currency,
			income_date, notes, is_deleted
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, FALSE)
		RETURNING created_at, updated_at, deleted_at
	`, income.ID, income.UserID, income.WalletID, income.CategoryID, income.SourceName,
		income.Amount, income.Currency, income.IncomeDate, income.Notes)

	var deletedAt sql.NullTime
	if err := row.Scan(&income.CreatedAt, &income.UpdatedAt, &deletedAt); err != nil {
		return err
	}
	if deletedAt.Valid {
		income.DeletedAt = &deletedAt.Time
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetIncomeByID(ctx context.Context, userID, incomeID uuid.UUID) (*Income, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, wallet_id, category_id, source_name, amount, currency,
			income_date, notes, is_deleted, created_at, updated_at, deleted_at
		FROM incomes
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
	`, userID, incomeID)

	income, err := scanIncome(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
		}
		return nil, err
	}

	return income, nil
}

func (r *Repository) ListIncomes(ctx context.Context, userID uuid.UUID, limit int, pagination *Pagination) ([]*Income, *Pagination, error) {
	query := `
		SELECT id, user_id, wallet_id, category_id, source_name, amount, currency,
			income_date, notes, is_deleted, created_at, updated_at, deleted_at
		FROM incomes
		WHERE user_id = $1 AND is_deleted = FALSE
	`

	args := []any{userID}
	if pagination != nil {
		query += " AND (created_at, id) < ($2, $3)"
		args = append(args, pagination.CreatedAt, pagination.ID)
	}

	args = append(args, limit+1)
	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	incomes := make([]*Income, 0, limit)
	for rows.Next() {
		income, err := scanIncome(rows)
		if err != nil {
			return nil, nil, err
		}
		incomes = append(incomes, income)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var nextPagination *Pagination
	if len(incomes) > limit {
		last := incomes[limit-1]
		nextPagination = &Pagination{CreatedAt: last.CreatedAt, ID: last.ID}
		incomes = incomes[:limit]
	}

	return incomes, nextPagination, nil
}

func (r *Repository) UpdateIncome(ctx context.Context, income *Income) error {
	if income == nil {
		return domain.NewDomainError(domain.ErrInternal, "income is required", 400)
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
		FROM incomes
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
		FOR UPDATE
	`, income.UserID, income.ID)
	if err := row.Scan(&oldWalletID, &oldAmount); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
		}
		return err
	}

	if oldWalletID == income.WalletID {
		balances, err := lockWalletBalances(ctx, tx, income.UserID, oldWalletID)
		if err != nil {
			return err
		}

		newBalance := balances[oldWalletID] - oldAmount + income.Amount
		if _, err := tx.Exec(ctx, `
			UPDATE wallets
			SET current_balance = $1,
				updated_at = NOW()
			WHERE id = $2 AND user_id = $3
		`, round2(newBalance), oldWalletID, income.UserID); err != nil {
			return err
		}
	} else {
		balances, err := lockWalletBalances(ctx, tx, income.UserID, oldWalletID, income.WalletID)
		if err != nil {
			return err
		}

		oldWalletBalance := balances[oldWalletID] - oldAmount
		newWalletBalance := balances[income.WalletID] + income.Amount

		if _, err := tx.Exec(ctx, `
			UPDATE wallets
			SET current_balance = $1,
				updated_at = NOW()
			WHERE id = $2 AND user_id = $3
		`, round2(oldWalletBalance), oldWalletID, income.UserID); err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, `
			UPDATE wallets
			SET current_balance = $1,
				updated_at = NOW()
			WHERE id = $2 AND user_id = $3
		`, round2(newWalletBalance), income.WalletID, income.UserID); err != nil {
			return err
		}
	}

	row = tx.QueryRow(ctx, `
		UPDATE incomes
		SET wallet_id = $3,
			category_id = $4,
			source_name = $5,
			amount = $6,
			currency = $7,
			income_date = $8,
			notes = $9,
			updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
		RETURNING created_at, updated_at, deleted_at
	`, income.UserID, income.ID, income.WalletID, income.CategoryID, income.SourceName,
		income.Amount, income.Currency, income.IncomeDate, income.Notes)

	var deletedAt sql.NullTime
	if err := row.Scan(&income.CreatedAt, &income.UpdatedAt, &deletedAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
		}
		return err
	}
	if deletedAt.Valid {
		income.DeletedAt = &deletedAt.Time
	} else {
		income.DeletedAt = nil
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (r *Repository) SoftDeleteIncome(ctx context.Context, userID, incomeID uuid.UUID) error {
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
		FROM incomes
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
		FOR UPDATE
	`, userID, incomeID)
	if err := row.Scan(&walletID, &amount); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
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
	`, round2(balances[walletID]-amount), walletID, userID); err != nil {
		return err
	}

	row = tx.QueryRow(ctx, `
		UPDATE incomes
		SET is_deleted = TRUE,
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND is_deleted = FALSE
		RETURNING id
	`, userID, incomeID)

	var id uuid.UUID
	if err := row.Scan(&id); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "income not found", 404)
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

func scanIncome(scanner interface{ Scan(...any) error }) (*Income, error) {
	income := &Income{}
	var categoryID sql.NullString
	var notes sql.NullString
	var deletedAt sql.NullTime

	err := scanner.Scan(
		&income.ID,
		&income.UserID,
		&income.WalletID,
		&categoryID,
		&income.SourceName,
		&income.Amount,
		&income.Currency,
		&income.IncomeDate,
		&notes,
		&income.IsDeleted,
		&income.CreatedAt,
		&income.UpdatedAt,
		&deletedAt,
	)
	if err != nil {
		return nil, err
	}

	if categoryID.Valid {
		id, err := uuid.Parse(categoryID.String)
		if err != nil {
			return nil, err
		}
		income.CategoryID = &id
	}
	if notes.Valid {
		income.Notes = &notes.String
	}
	if deletedAt.Valid {
		income.DeletedAt = &deletedAt.Time
	}

	return income, nil
}
