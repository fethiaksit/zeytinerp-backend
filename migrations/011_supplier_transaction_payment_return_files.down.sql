DROP INDEX IF EXISTS idx_supplier_transactions_image_url;

ALTER TABLE supplier_transactions
DROP COLUMN IF EXISTS file_path,
DROP COLUMN IF EXISTS image_url;
