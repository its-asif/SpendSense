package wallet

import (
    "context"
    "database/sql"

    "spendsense-backend/internal/domain"
    "spendsense-backend/internal/infra"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

type Repository struct {
    db *infra.Database
}

func NewRepository(db *infra.Database) *Repository { return &Repository{db: db} }

func (r *Repository) CreateWallet(ctx context.Context, w *Wallet) error {
    if w == nil {
        return domain.NewDomainError(domain.ErrInternal, "wallet required", 400)
    }

    row := r.db.QueryRow(ctx, `
        INSERT INTO wallets (id, user_id, name, wallet_type, provider, account_number, account_name, currency, opening_balance, current_balance, is_active)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, TRUE)
        RETURNING created_at, updated_at
    `, w.ID, w.UserID, w.Name, w.WalletType, w.Provider, w.AccountNumber, w.AccountName, w.Currency, w.OpeningBalance, w.CurrentBalance)

    if err := row.Scan(&w.CreatedAt, &w.UpdatedAt); err != nil {
        return err
    }
    return nil
}

func (r *Repository) GetWalletByID(ctx context.Context, userID, id uuid.UUID) (*Wallet, error) {
    row := r.db.QueryRow(ctx, `
        SELECT id, user_id, name, wallet_type, provider, account_number, account_name, currency, opening_balance, current_balance, is_active, created_at, updated_at
        FROM wallets WHERE id = $1 AND user_id = $2
    `, id, userID)

    w := &Wallet{}
    var provider sql.NullString
    var accountNumber sql.NullString
    var accountName sql.NullString
    if err := row.Scan(&w.ID, &w.UserID, &w.Name, &w.WalletType, &provider, &accountNumber, &accountName, &w.Currency, &w.OpeningBalance, &w.CurrentBalance, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
        if err == pgx.ErrNoRows {
            return nil, domain.NewDomainError(domain.ErrNotFound, "wallet not found", 404)
        }
        return nil, err
    }
    if provider.Valid { v := provider.String; w.Provider = &v }
    if accountNumber.Valid { v := accountNumber.String; w.AccountNumber = &v }
    if accountName.Valid { v := accountName.String; w.AccountName = &v }
    return w, nil
}

func (r *Repository) ListWallets(ctx context.Context, userID uuid.UUID) ([]*Wallet, error) {
    rows, err := r.db.Query(ctx, `
        SELECT id, user_id, name, wallet_type, provider, account_number, account_name, currency, opening_balance, current_balance, is_active, created_at, updated_at
        FROM wallets WHERE user_id = $1
        ORDER BY created_at DESC
    `, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    list := []*Wallet{}
    for rows.Next() {
        w := &Wallet{}
        var provider sql.NullString
        var accountNumber sql.NullString
        var accountName sql.NullString
        if err := rows.Scan(&w.ID, &w.UserID, &w.Name, &w.WalletType, &provider, &accountNumber, &accountName, &w.Currency, &w.OpeningBalance, &w.CurrentBalance, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
            return nil, err
        }
        if provider.Valid { v := provider.String; w.Provider = &v }
        if accountNumber.Valid { v := accountNumber.String; w.AccountNumber = &v }
        if accountName.Valid { v := accountName.String; w.AccountName = &v }
        list = append(list, w)
    }
    return list, nil
}

func (r *Repository) UpdateWallet(ctx context.Context, w *Wallet) error {
    row := r.db.QueryRow(ctx, `
        UPDATE wallets SET name=$3, wallet_type=$4, provider=$5, account_number=$6, account_name=$7, currency=$8, current_balance=$9, is_active=$10, updated_at=NOW()
        WHERE user_id=$1 AND id=$2
        RETURNING created_at, updated_at
    `, w.UserID, w.ID, w.Name, w.WalletType, w.Provider, w.AccountNumber, w.AccountName, w.Currency, w.CurrentBalance, w.IsActive)

    if err := row.Scan(&w.CreatedAt, &w.UpdatedAt); err != nil {
        if err == pgx.ErrNoRows {
            return domain.NewDomainError(domain.ErrNotFound, "wallet not found", 404)
        }
        return err
    }
    return nil
}

func (r *Repository) DeleteWallet(ctx context.Context, userID, id uuid.UUID) error {
    tag, err := r.db.Exec(ctx, `DELETE FROM wallets WHERE user_id=$1 AND id=$2`, userID, id)
    if err != nil {
        return err
    }
    if tag.RowsAffected() == 0 {
        return domain.NewDomainError(domain.ErrNotFound, "wallet not found", 404)
    }
    return nil
}
