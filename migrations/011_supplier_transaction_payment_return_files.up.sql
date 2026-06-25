ALTER TABLE supplier_transactions
ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS file_path TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_supplier_transactions_image_url
ON supplier_transactions(image_url);
