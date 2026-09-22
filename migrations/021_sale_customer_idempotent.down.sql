DROP INDEX IF EXISTS idx_customer_transactions_sale_id;
ALTER TABLE customer_transactions DROP COLUMN IF EXISTS sale_id;

DROP INDEX IF EXISTS idx_sales_customer_id;
ALTER TABLE sales DROP COLUMN IF EXISTS customer_id;
