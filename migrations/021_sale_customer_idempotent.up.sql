-- Idempotent Migration: sales tablosuna customer_id ve customer_transactions tablosuna sale_id ekleme
ALTER TABLE sales ADD COLUMN IF NOT EXISTS customer_id BIGINT REFERENCES customers(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_sales_customer_id ON sales (customer_id);

ALTER TABLE customer_transactions ADD COLUMN IF NOT EXISTS sale_id BIGINT REFERENCES sales(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_customer_transactions_sale_id ON customer_transactions (sale_id);
