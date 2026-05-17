-- 003_add_budgets_wallets_and_reporting.up.sql

-- Budgets table
CREATE TABLE budgets (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
	amount NUMERIC(18,2) NOT NULL CHECK (amount > 0),
	currency VARCHAR(3) NOT NULL DEFAULT 'USD',
	period VARCHAR(20) NOT NULL DEFAULT 'MONTHLY', -- MONTHLY, YEARLY, WEEKLY
	start_date DATE NOT NULL,
	rollover_enabled BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_budgets_user_id ON budgets(user_id);
CREATE INDEX idx_budgets_category_id ON budgets(category_id);

-- Tags table
CREATE TABLE tags (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name VARCHAR(30) NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tags_user_id ON tags(user_id);
CREATE UNIQUE INDEX idx_tags_name_user_id ON tags(user_id, name);

-- Join table for expenses and tags
CREATE TABLE expense_tags (
	expense_id UUID NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
	tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
	PRIMARY KEY (expense_id, tag_id)
);

CREATE INDEX idx_expense_tags_tag_id ON expense_tags(tag_id);

-- Receipts table
CREATE TABLE receipts (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	expense_id UUID NOT NULL UNIQUE REFERENCES expenses(id) ON DELETE CASCADE,
	file_path VARCHAR(500) NOT NULL,
	file_size BIGINT NOT NULL,
	mime_type VARCHAR(50) NOT NULL,
	uploaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP
);

CREATE INDEX idx_receipts_expense_id ON receipts(expense_id);

-- Notifications table
CREATE TABLE notifications (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	type VARCHAR(50) NOT NULL, -- budget_alert, recurring_created, anomaly_detected
	title VARCHAR(255) NOT NULL,
	body TEXT NOT NULL,
	metadata JSONB,
	is_read BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	dismissed_at TIMESTAMP
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_is_read ON notifications(is_read);

-- Budget alerts tracking (to avoid duplicate alerts)
CREATE TABLE budget_alerts_sent (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	budget_id UUID NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
	threshold_percent INTEGER NOT NULL, -- 75, 90, 100
	sent_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	year_month VARCHAR(7) NOT NULL, -- YYYY-MM
	UNIQUE(budget_id, threshold_percent, year_month)
);

CREATE INDEX idx_budget_alerts_sent_budget_id ON budget_alerts_sent(budget_id);

-- Wallet transaction fees for transfer-cost breakdown and reporting
CREATE TABLE wallet_transfer_fees (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	transfer_id UUID NOT NULL REFERENCES wallet_transfers(id) ON DELETE CASCADE,
	fee_type VARCHAR(20) NOT NULL DEFAULT 'SERVICE', -- SERVICE, VAT, OTHER
	fee_amount NUMERIC(18,2) NOT NULL CHECK (fee_amount >= 0),
	notes TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_wallet_transfer_fees_transfer_id ON wallet_transfer_fees(transfer_id);
