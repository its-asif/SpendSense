-- 001_init_schema.up.sql

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Users table
CREATE TABLE users (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	email VARCHAR(255) NOT NULL UNIQUE,
	password_hash VARCHAR(255) NOT NULL,
	display_name VARCHAR(255),
	avatar_path VARCHAR(500),
	base_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
	timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
	locale VARCHAR(10) NOT NULL DEFAULT 'en-US',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);

-- Categories table
CREATE TABLE categories (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID REFERENCES users(id) ON DELETE CASCADE,
	name VARCHAR(100) NOT NULL,
	icon VARCHAR(10),
	color VARCHAR(7),
	is_default BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_categories_user_id ON categories(user_id);
CREATE UNIQUE INDEX idx_categories_name_user_id ON categories(user_id, name);

-- Wallets table
CREATE TABLE wallets (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name VARCHAR(100) NOT NULL,
	wallet_type VARCHAR(20) NOT NULL, -- CASH, MOBILE_WALLET, BANK, CARD
	provider VARCHAR(100),
	account_number VARCHAR(100),      -- optional: account number for provider
	account_name VARCHAR(200),        -- optional: account display name (e.g., "Savings XXXX")
	currency VARCHAR(3) NOT NULL DEFAULT 'USD',
	opening_balance NUMERIC(18,2) NOT NULL DEFAULT 0,
	current_balance NUMERIC(18,2) NOT NULL DEFAULT 0,
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(user_id, name)
);

CREATE INDEX idx_wallets_user_id ON wallets(user_id);
CREATE INDEX idx_wallets_type ON wallets(wallet_type);
CREATE INDEX idx_wallets_account_number ON wallets(account_number);
CREATE UNIQUE INDEX idx_wallets_user_provider_accountnum ON wallets(user_id, provider, account_number);

-- Expenses table
CREATE TABLE expenses (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	wallet_id UUID NOT NULL REFERENCES wallets(id),
	amount NUMERIC(18,2) NOT NULL CHECK (amount > 0),
	currency VARCHAR(3) NOT NULL DEFAULT 'USD',
	fx_rate_to_base NUMERIC(18,8) NOT NULL DEFAULT 1.0,
	category_id UUID NOT NULL REFERENCES categories(id),
	merchant VARCHAR(255),
	date DATE NOT NULL,
	notes TEXT,
	is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
	recurring_rule VARCHAR(50),
	is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP
);

CREATE INDEX idx_expenses_user_id ON expenses(user_id);
CREATE INDEX idx_expenses_user_date ON expenses(user_id, date);
CREATE INDEX idx_expenses_category_id ON expenses(category_id);
CREATE INDEX idx_expenses_wallet_id ON expenses(wallet_id);
CREATE INDEX idx_expenses_is_deleted ON expenses(is_deleted);

-- Income table
CREATE TABLE incomes (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	wallet_id UUID NOT NULL REFERENCES wallets(id),
	category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
	source_name VARCHAR(255) NOT NULL,
	amount NUMERIC(18,2) NOT NULL CHECK (amount > 0),
	currency VARCHAR(3) NOT NULL DEFAULT 'USD',
	income_date DATE NOT NULL,
	notes TEXT,
	is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP
);

CREATE INDEX idx_incomes_user_id ON incomes(user_id);
CREATE INDEX idx_incomes_wallet_id ON incomes(wallet_id);
CREATE INDEX idx_incomes_income_date ON incomes(income_date);
CREATE INDEX idx_incomes_category_id ON incomes(category_id);

-- Wallet transfers table
CREATE TABLE wallet_transfers (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	from_wallet_id UUID NOT NULL REFERENCES wallets(id),
	to_wallet_id UUID NOT NULL REFERENCES wallets(id),
	amount NUMERIC(18,2) NOT NULL CHECK (amount > 0),
	fee_amount NUMERIC(18,2) NOT NULL DEFAULT 0 CHECK (fee_amount >= 0),
	currency VARCHAR(3) NOT NULL DEFAULT 'USD',
	transfer_date DATE NOT NULL,
	notes TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CHECK (from_wallet_id <> to_wallet_id)
);

CREATE INDEX idx_wallet_transfers_user_id ON wallet_transfers(user_id);
CREATE INDEX idx_wallet_transfers_from_wallet_id ON wallet_transfers(from_wallet_id);
CREATE INDEX idx_wallet_transfers_to_wallet_id ON wallet_transfers(to_wallet_id);

-- Audit logs table
CREATE TABLE audit_logs (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	entity_type VARCHAR(50) NOT NULL,
	entity_id UUID NOT NULL,
	action VARCHAR(20) NOT NULL,
	before JSONB,
	after JSONB,
	ip_address VARCHAR(45),
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_user_entity ON audit_logs(user_id, entity_type, entity_id);

-- Refresh tokens table (Redis alternative: use Redis with TTL)
CREATE TABLE refresh_tokens (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token_hash VARCHAR(255) NOT NULL UNIQUE,
	expires_at TIMESTAMP NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
