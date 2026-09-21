-- ZeytinERP - Cari Müşteri İyileştirmeleri ve İndeksleri Migration
ALTER TABLE customers ADD COLUMN IF NOT EXISTS customer_type TEXT NOT NULL DEFAULT 'cari';
ALTER TABLE customers ALTER COLUMN customer_type SET DEFAULT 'cari';
UPDATE customers SET customer_type = 'cari' WHERE customer_type IS NULL OR customer_type = '' OR customer_type = 'normal';

ALTER TABLE customers ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS credit_limit NUMERIC(12,2) DEFAULT NULL;

CREATE INDEX IF NOT EXISTS idx_customers_phone ON customers (phone) WHERE phone IS NOT NULL AND phone <> '';
CREATE INDEX IF NOT EXISTS idx_customer_transactions_customer_id ON customer_transactions (customer_id);
CREATE INDEX IF NOT EXISTS idx_customer_transactions_date ON customer_transactions (transaction_date);
