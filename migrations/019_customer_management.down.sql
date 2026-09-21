DROP INDEX IF EXISTS idx_customers_phone;
ALTER TABLE customers DROP COLUMN IF EXISTS credit_limit;
ALTER TABLE customers DROP COLUMN IF EXISTS is_active;
