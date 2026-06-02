CREATE TABLE IF NOT EXISTS financial_debts (
    id BIGSERIAL PRIMARY KEY,
    debt_type TEXT NOT NULL CHECK (debt_type IN ('bank_loan', 'credit_card', 'installment_debt', 'other')),
    institution_name TEXT NOT NULL,
    title TEXT NOT NULL,
    total_amount NUMERIC(12,2) NOT NULL CHECK (total_amount > 0),
    installment_count INTEGER NOT NULL DEFAULT 0 CHECK (installment_count >= 0),
    installment_amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (installment_amount >= 0),
    start_date DATE NOT NULL,
    due_date DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'closed')),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_financial_debts_due_date ON financial_debts(due_date);
CREATE INDEX IF NOT EXISTS idx_financial_debts_status ON financial_debts(status);
CREATE INDEX IF NOT EXISTS idx_financial_debts_debt_type ON financial_debts(debt_type);

CREATE TABLE IF NOT EXISTS financial_debt_payments (
    id BIGSERIAL PRIMARY KEY,
    financial_debt_id BIGINT NOT NULL REFERENCES financial_debts(id) ON DELETE CASCADE,
    payment_date DATE NOT NULL,
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    payment_method TEXT NOT NULL CHECK (payment_method IN ('cash', 'bank_transfer', 'credit_card', 'other')),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_financial_debt_payments_debt_id ON financial_debt_payments(financial_debt_id);
CREATE INDEX IF NOT EXISTS idx_financial_debt_payments_date ON financial_debt_payments(payment_date);
