DROP INDEX IF EXISTS idx_exchange_rates_latest;
DROP INDEX IF EXISTS idx_exchange_rates_currency_source_date;
DROP TABLE IF EXISTS exchange_rates;

ALTER TABLE supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_amount_try_check;
ALTER TABLE supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_amount_original_check;
ALTER TABLE supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_exchange_rate_check;
ALTER TABLE supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_currency_check;
DROP INDEX IF EXISTS idx_supplier_transactions_amount_try;
DROP INDEX IF EXISTS idx_supplier_transactions_currency;
ALTER TABLE supplier_transactions DROP COLUMN IF EXISTS amount_try;
ALTER TABLE supplier_transactions DROP COLUMN IF EXISTS amount_original;
ALTER TABLE supplier_transactions DROP COLUMN IF EXISTS exchange_rate;
ALTER TABLE supplier_transactions DROP COLUMN IF EXISTS currency;
