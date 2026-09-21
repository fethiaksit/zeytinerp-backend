-- ZeytinERP - Cari Müşteri Yönetimi Migration
ALTER TABLE customers ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE customers ADD COLUMN IF NOT EXISTS credit_limit NUMERIC(12,2) DEFAULT NULL;

-- Telefon numarası benzersizlik indeksi (boş olmayanlar için)
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_phone ON customers (phone) WHERE phone IS NOT NULL AND phone <> '';
