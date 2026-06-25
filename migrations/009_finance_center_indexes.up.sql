CREATE INDEX IF NOT EXISTS idx_supplier_transactions_type_date
    ON supplier_transactions(type, transaction_date);

CREATE INDEX IF NOT EXISTS idx_employee_transactions_type_date
    ON employee_transactions(type, transaction_date);

CREATE INDEX IF NOT EXISTS idx_financial_debt_payments_date_debt
    ON financial_debt_payments(payment_date, financial_debt_id);

CREATE INDEX IF NOT EXISTS idx_financial_debt_installments_debt_due
    ON financial_debt_installments(financial_debt_id, due_date);
