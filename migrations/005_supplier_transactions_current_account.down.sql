DROP INDEX IF EXISTS idx_supplier_transactions_invoice_no;
DROP INDEX IF EXISTS idx_supplier_transactions_type;

ALTER TABLE supplier_transactions DROP CONSTRAINT IF EXISTS supplier_transactions_type_check;
ALTER TABLE supplier_transactions
ADD CONSTRAINT supplier_transactions_type_check
CHECK (type IN ('purchase', 'payment'));

ALTER TABLE supplier_transactions DROP COLUMN IF EXISTS invoice_no;
