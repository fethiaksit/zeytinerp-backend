ALTER TABLE financial_debt_payments DROP CONSTRAINT IF EXISTS fk_financial_debt_payments_installment;
DROP INDEX IF EXISTS idx_financial_debt_payments_installment_id;
ALTER TABLE financial_debt_payments DROP COLUMN IF EXISTS installment_id;
DROP TABLE IF EXISTS financial_debt_installments;
ALTER TABLE financial_debts DROP COLUMN IF EXISTS end_date;
