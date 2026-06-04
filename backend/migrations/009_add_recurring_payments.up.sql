CREATE TABLE recurring_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    amount NUMERIC(18,2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    interval VARCHAR(20) NOT NULL DEFAULT 'monthly', -- daily, weekly, monthly, yearly
    start_date DATE NOT NULL,
    deadline DATE NOT NULL,
    alert_rule VARCHAR(10) NOT NULL DEFAULT '1d', -- start, 1d, 7d, 1h, 12h
    end_date DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'unpaid', -- unpaid, paid, inactive
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_recurring_payments_user_id ON recurring_payments(user_id);
CREATE INDEX idx_recurring_payments_wallet_id ON recurring_payments(wallet_id);
CREATE INDEX idx_recurring_payments_status ON recurring_payments(status);
