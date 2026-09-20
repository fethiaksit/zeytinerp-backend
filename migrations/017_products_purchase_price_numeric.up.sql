-- 017_products_purchase_price_numeric.up.sql
-- Allow decimal purchase prices (e.g. 17.60) without deleting or resetting data.
ALTER TABLE products
  ALTER COLUMN purchase_price TYPE NUMERIC(12,2)
  USING purchase_price::NUMERIC(12,2);
