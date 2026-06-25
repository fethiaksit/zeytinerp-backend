ALTER TABLE supplier_transactions
    ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'TRY',
    ADD COLUMN IF NOT EXISTS exchange_rate NUMERIC(18,6) NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS amount_original NUMERIC(14,2),
    ADD COLUMN IF NOT EXISTS amount_try NUMERIC(14,2);

UPDATE supplier_transactions
SET
    currency = 'TRY',
    exchange_rate = 1,
    amount_original = amount,
    amount_try = amount
WHERE amount_original IS NULL OR amount_try IS NULL;

ALTER TABLE supplier_transactions
    ALTER COLUMN amount_original SET NOT NULL,
    ALTER COLUMN amount_try SET NOT NULL;

ALTER TABLE supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_currency_check;
ALTER TABLE supplier_transactions ADD CONSTRAINT supplier_transactions_currency_check
    CHECK (currency IN ('TRY', 'USD', 'EUR'));

ALTER TABLE supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_exchange_rate_check;
ALTER TABLE supplier_transactions ADD CONSTRAINT supplier_transactions_exchange_rate_check
    CHECK (exchange_rate > 0);

ALTER TABLE supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_amount_original_check;
ALTER TABLE supplier_transactions ADD CONSTRAINT supplier_transactions_amount_original_check
    CHECK (amount_original > 0);

ALTER TABLE supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_amount_try_check;
ALTER TABLE supplier_transactions ADD CONSTRAINT supplier_transactions_amount_try_check
    CHECK (amount_try > 0);

CREATE INDEX IF NOT EXISTS idx_supplier_transactions_currency ON supplier_transactions(currency);
CREATE INDEX IF NOT EXISTS idx_supplier_transactions_amount_try ON supplier_transactions(amount_try);

CREATE TABLE IF NOT EXISTS exchange_rates (
    id BIGSERIAL PRIMARY KEY,
    currency_code TEXT NOT NULL CHECK (currency_code IN ('USD', 'EUR')),
    rate_to_try NUMERIC(18,6) NOT NULL CHECK (rate_to_try > 0),
    source TEXT NOT NULL,
    rate_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_exchange_rates_currency_source_date
    ON exchange_rates(currency_code, source, rate_date);
CREATE INDEX IF NOT EXISTS idx_exchange_rates_latest
    ON exchange_rates(currency_code, rate_date DESC, created_at DESC);
