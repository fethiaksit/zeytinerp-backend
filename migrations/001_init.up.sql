CREATE TABLE IF NOT EXISTS suppliers (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    phone TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

CREATE TABLE IF NOT EXISTS supplier_transactions (
    id BIGSERIAL PRIMARY KEY,
    supplier_id BIGINT NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    transaction_date DATE NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('purchase', 'payment')),
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    payment_method TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE supplier_transactions ADD COLUMN IF NOT EXISTS payment_method TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_supplier_transactions_supplier_id ON supplier_transactions(supplier_id);
CREATE INDEX IF NOT EXISTS idx_supplier_transactions_date ON supplier_transactions(transaction_date);

CREATE TABLE IF NOT EXISTS employees (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    phone TEXT NOT NULL DEFAULT '',
    daily_wage NUMERIC(12,2) NOT NULL CHECK (daily_wage > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS employee_transactions (
    id BIGSERIAL PRIMARY KEY,
    employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    transaction_date DATE NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('work', 'payment', 'advance')),
    work_days NUMERIC(8,2) NOT NULL DEFAULT 0 CHECK (work_days >= 0),
    amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (amount >= 0),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (type = 'work' AND work_days > 0)
        OR (type IN ('payment', 'advance') AND amount > 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_employee_transactions_employee_id ON employee_transactions(employee_id);
CREATE INDEX IF NOT EXISTS idx_employee_transactions_date ON employee_transactions(transaction_date);

CREATE TABLE IF NOT EXISTS daily_cash_reports (
    id BIGSERIAL PRIMARY KEY,
    report_date DATE NOT NULL,
    cash_amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (cash_amount >= 0),
    pos_amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (pos_amount >= 0),
    qr_amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (qr_amount >= 0),
    credit_collection NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (credit_collection >= 0),
    credit_given NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (credit_given >= 0),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_daily_cash_reports_report_date ON daily_cash_reports(report_date);

CREATE TABLE IF NOT EXISTS expenses (
    id BIGSERIAL PRIMARY KEY,
    expense_date DATE NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('kira', 'elektrik', 'su', 'personel', 'yakit', 'yemek', 'market_gideri', 'diger')),
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    payment_method TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE expenses ADD COLUMN IF NOT EXISTS payment_method TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_expenses_expense_date ON expenses(expense_date);

CREATE TABLE IF NOT EXISTS income_entries (
    id BIGSERIAL PRIMARY KEY,
    income_date DATE NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('market_satis', 'tup_satis', 'veresiye_tahsilat', 'diger')),
    amount NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    payment_method TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_income_entries_income_date ON income_entries(income_date);

CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    barcode TEXT UNIQUE,
    category TEXT NOT NULL DEFAULT '',
    purchase_price NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (purchase_price >= 0),
    sale_price NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (sale_price >= 0),
    critical_stock NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (critical_stock >= 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stock_movements (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    movement_date DATE NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('in', 'out', 'waste', 'correction')),
    quantity NUMERIC(12,2) NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_product_id ON stock_movements(product_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_date ON stock_movements(movement_date);
