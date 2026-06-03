CREATE TABLE IF NOT EXISTS supplier_transaction_files (
    id BIGSERIAL PRIMARY KEY,
    supplier_transaction_id BIGINT NOT NULL REFERENCES supplier_transactions(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_url TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size BIGINT NOT NULL CHECK (size > 0),
    page_order INTEGER NOT NULL DEFAULT 0 CHECK (page_order >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_transaction_files_transaction_id
ON supplier_transaction_files(supplier_transaction_id);

CREATE INDEX IF NOT EXISTS idx_supplier_transaction_files_page_order
ON supplier_transaction_files(supplier_transaction_id, page_order);
