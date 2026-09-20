-- 015_categories.up.sql
-- Non-destructive category master table for POS/product imports.
CREATE TABLE IF NOT EXISTS categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_categories_name_normalized
    ON categories (LOWER(BTRIM(name)));

-- Preserve and register all category names already used by products.
INSERT INTO categories (name)
SELECT DISTINCT BTRIM(category)
FROM products
WHERE category IS NOT NULL
  AND BTRIM(category) <> ''
ON CONFLICT DO NOTHING;
