CREATE INDEX IF NOT EXISTS idx_supplier_transactions_cash_payment_date
    ON supplier_transactions(transaction_date)
    WHERE type = 'payment'
      AND LOWER(BTRIM(payment_method)) IN ('cash', 'nakit');
