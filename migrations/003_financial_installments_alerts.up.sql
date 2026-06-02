ALTER TABLE financial_debts ADD COLUMN IF NOT EXISTS end_date DATE;
UPDATE financial_debts
SET end_date = COALESCE(end_date, due_date, start_date, CURRENT_DATE)
WHERE end_date IS NULL;
ALTER TABLE financial_debts ALTER COLUMN end_date SET NOT NULL;

CREATE TABLE IF NOT EXISTS financial_debt_installments (
    id BIGSERIAL PRIMARY KEY,
    financial_debt_id BIGINT NOT NULL REFERENCES financial_debts(id) ON DELETE CASCADE,
    installment_no INTEGER NOT NULL CHECK (installment_no > 0),
    due_date DATE NOT NULL,
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    paid_amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'partial_paid', 'paid', 'overdue')),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_financial_debt_installments_debt_id ON financial_debt_installments(financial_debt_id);
CREATE INDEX IF NOT EXISTS idx_financial_debt_installments_due_date ON financial_debt_installments(due_date);
CREATE INDEX IF NOT EXISTS idx_financial_debt_installments_status ON financial_debt_installments(status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_financial_debt_installments_unique_no ON financial_debt_installments(financial_debt_id, installment_no);

ALTER TABLE financial_debt_payments ADD COLUMN IF NOT EXISTS installment_id BIGINT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_financial_debt_payments_installment'
    ) THEN
        ALTER TABLE financial_debt_payments
        ADD CONSTRAINT fk_financial_debt_payments_installment
        FOREIGN KEY (installment_id)
        REFERENCES financial_debt_installments(id)
        ON DELETE CASCADE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_financial_debt_payments_installment_id ON financial_debt_payments(installment_id);
