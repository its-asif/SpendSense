-- 004_add_income_and_loan_management.up.sql

-- Personal loan tracking table (lend/borrow between people)
CREATE TABLE personal_loans (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	wallet_id UUID NOT NULL REFERENCES wallets(id),
	loan_direction VARCHAR(10) NOT NULL CHECK (loan_direction IN ('LENT', 'BORROWED')),
	counterparty_name VARCHAR(255) NOT NULL,
	counterparty_email VARCHAR(255),
	principal_amount NUMERIC(18,2) NOT NULL CHECK (principal_amount > 0),
	currency VARCHAR(3) NOT NULL DEFAULT 'USD',
	loan_date DATE NOT NULL,
	due_date DATE,
	reminder_at TIMESTAMP,
	snooze_until TIMESTAMP,
	interest_amount NUMERIC(18,2) NOT NULL DEFAULT 0 CHECK (interest_amount >= 0),
	status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'PARTIAL', 'SETTLED', 'OVERDUE', 'CANCELLED')),
	notes TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	closed_at TIMESTAMP
);

CREATE INDEX idx_personal_loans_user_id ON personal_loans(user_id);
CREATE INDEX idx_personal_loans_wallet_id ON personal_loans(wallet_id);
CREATE INDEX idx_personal_loans_counterparty_email ON personal_loans(counterparty_email);
CREATE INDEX idx_personal_loans_due_date ON personal_loans(due_date);

-- Personal loan payment history
CREATE TABLE personal_loan_payments (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	personal_loan_id UUID NOT NULL REFERENCES personal_loans(id) ON DELETE CASCADE,
	payment_date DATE NOT NULL,
	amount NUMERIC(18,2) NOT NULL CHECK (amount > 0),
	payment_type VARCHAR(20) NOT NULL DEFAULT 'REPAYMENT' CHECK (payment_type IN ('REPAYMENT', 'PARTIAL', 'INTEREST', 'ADJUSTMENT')),
	notes TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_personal_loan_payments_loan_id ON personal_loan_payments(personal_loan_id);

-- Bank loan management
CREATE TABLE bank_loans (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	wallet_id UUID NOT NULL REFERENCES wallets(id),
	bank_name VARCHAR(255) NOT NULL,
	loan_account_number VARCHAR(100),
	principal_amount NUMERIC(18,2) NOT NULL CHECK (principal_amount > 0),
	currency VARCHAR(3) NOT NULL DEFAULT 'USD',
	disbursement_date DATE,
	repayment_start_date DATE NOT NULL,
	annual_interest_rate NUMERIC(7,4) NOT NULL CHECK (annual_interest_rate >= 0),
	term_months INTEGER NOT NULL CHECK (term_months > 0),
	payment_day_of_month INTEGER NOT NULL DEFAULT 1 CHECK (payment_day_of_month BETWEEN 1 AND 28),
	monthly_installment NUMERIC(18,2) NOT NULL CHECK (monthly_installment > 0),
	total_interest_amount NUMERIC(18,2) NOT NULL DEFAULT 0 CHECK (total_interest_amount >= 0),
	total_payable_amount NUMERIC(18,2) NOT NULL DEFAULT 0 CHECK (total_payable_amount >= 0),
	status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'CLOSED', 'DEFAULTED', 'PAUSED')),
	notes TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	closed_at TIMESTAMP
);

CREATE INDEX idx_bank_loans_user_id ON bank_loans(user_id);
CREATE INDEX idx_bank_loans_wallet_id ON bank_loans(wallet_id);
CREATE INDEX idx_bank_loans_repayment_start_date ON bank_loans(repayment_start_date);

-- Bank loan repayment schedule and payment history
CREATE TABLE bank_loan_payments (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	bank_loan_id UUID NOT NULL REFERENCES bank_loans(id) ON DELETE CASCADE,
	installment_number INTEGER NOT NULL,
	due_date DATE NOT NULL,
	opening_balance NUMERIC(18,2) NOT NULL CHECK (opening_balance >= 0),
	principal_amount NUMERIC(18,2) NOT NULL CHECK (principal_amount >= 0),
	interest_amount NUMERIC(18,2) NOT NULL CHECK (interest_amount >= 0),
	fee_amount NUMERIC(18,2) NOT NULL DEFAULT 0 CHECK (fee_amount >= 0),
	total_amount NUMERIC(18,2) NOT NULL CHECK (total_amount > 0),
	closing_balance NUMERIC(18,2) NOT NULL CHECK (closing_balance >= 0),
	payment_status VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED' CHECK (payment_status IN ('SCHEDULED', 'PAID', 'SNOOZED', 'SKIPPED', 'LATE')),
	paid_at TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(bank_loan_id, installment_number)
);

CREATE INDEX idx_bank_loan_payments_loan_id ON bank_loan_payments(bank_loan_id);
CREATE INDEX idx_bank_loan_payments_due_date ON bank_loan_payments(due_date);