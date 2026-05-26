package wallet

import (
	"context"

	"spendsense-backend/internal/domain"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) CreateTransfer(ctx context.Context, t *Transfer) error {
	if t == nil {
		return domain.NewDomainError(domain.ErrInternal, "transfer required", 400)
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var fromBalance float64
	row := tx.QueryRow(ctx, `SELECT current_balance FROM wallets WHERE id=$1 AND user_id=$2 FOR UPDATE`, t.FromWalletID, t.UserID)
	if err := row.Scan(&fromBalance); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "from wallet not found", 404)
		}
		return err
	}

	var toBalance float64
	row = tx.QueryRow(ctx, `SELECT current_balance FROM wallets WHERE id=$1 AND user_id=$2 FOR UPDATE`, t.ToWalletID, t.UserID)
	if err := row.Scan(&toBalance); err != nil {
		if err == pgx.ErrNoRows {
			return domain.NewDomainError(domain.ErrNotFound, "to wallet not found", 404)
		}
		return err
	}

	// Ensure different wallets
	if t.FromWalletID == t.ToWalletID {
		return domain.NewDomainError(domain.ErrInvalidWallet, "from and to wallets must differ", 400)
	}

	// Ensure sufficient funds (including fee)
	if fromBalance < t.Amount+t.FeeAmount {
		return domain.NewDomainError(domain.ErrInvalidAmount, "insufficient funds", 400)
	}

	// Update balances
	if _, err := tx.Exec(ctx, `UPDATE wallets SET current_balance = current_balance - $1, updated_at=NOW() WHERE id=$2`, t.Amount+t.FeeAmount, t.FromWalletID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE wallets SET current_balance = current_balance + $1, updated_at=NOW() WHERE id=$2`, t.ConvertedAmount, t.ToWalletID); err != nil {
		return err
	}

	// Insert transfer record
	row = tx.QueryRow(ctx, `
        INSERT INTO wallet_transfers (id, user_id, from_wallet_id, to_wallet_id, amount, fee_amount, currency, transfer_date, notes)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
        RETURNING created_at
    `, t.ID, t.UserID, t.FromWalletID, t.ToWalletID, t.Amount, t.FeeAmount, t.Currency, t.TransferDate.Format("2006-01-02"), t.Notes)

	if err := row.Scan(&t.CreatedAt); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
