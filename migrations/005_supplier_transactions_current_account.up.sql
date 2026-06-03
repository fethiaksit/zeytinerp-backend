ALTER TABLE supplier_transactions ADD COLUMN IF NOT EXISTS invoice_no TEXT NOT NULL DEFAULT '';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'supplier_transactions_type_check'
          AND conrelid = 'supplier_transactions'::regclass
    ) THEN
        ALTER TABLE supplier_transactions DROP CONSTRAINT supplier_transactions_type_check;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'supplier_transactions_type_check'
          AND conrelid = 'supplier_transactions'::regclass
    ) THEN
        ALTER TABLE supplier_transactions
        ADD CONSTRAINT supplier_transactions_type_check
        CHECK (type IN ('purchase', 'invoice', 'payment', 'return'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_supplier_transactions_type ON supplier_transactions(type);
CREATE INDEX IF NOT EXISTS idx_supplier_transactions_invoice_no ON supplier_transactions(invoice_no);
