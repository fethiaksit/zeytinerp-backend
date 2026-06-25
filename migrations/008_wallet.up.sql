CREATE TABLE IF NOT EXISTS wallet_transactions (
    id BIGSERIAL PRIMARY KEY,
    transaction_date DATE NOT NULL,
    transaction_type TEXT NOT NULL CHECK (
        transaction_type IN (
            'opening_balance',
            'cash_income',
            'cash_sale',
            'pos_income',
            'bank_income',
            'payment',
            'expense',
            'cash_withdraw',
            'cash_deposit',
            'correction'
        )
    ),
    amount NUMERIC(12,2) NOT NULL,
    balance_after NUMERIC(12,2) NOT NULL DEFAULT 0,
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    related_type TEXT NOT NULL DEFAULT '',
    related_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (transaction_type = 'correction' AND amount <> 0)
        OR (transaction_type <> 'correction' AND amount > 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_wallet_transactions_date ON wallet_transactions(transaction_date);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_type ON wallet_transactions(transaction_type);
CREATE INDEX IF NOT EXISTS idx_wallet_transactions_related ON wallet_transactions(related_type, related_id);
