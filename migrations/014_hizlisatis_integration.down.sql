DROP TABLE IF EXISTS sale_items;
DROP TABLE IF EXISTS sales;
DROP TABLE IF EXISTS users;

DROP INDEX IF EXISTS idx_products_bestseller;

ALTER TABLE products DROP COLUMN IF EXISTS bestseller_order;
ALTER TABLE products DROP COLUMN IF EXISTS is_bestseller;
ALTER TABLE products DROP COLUMN IF EXISTS image_url;
ALTER TABLE products DROP COLUMN IF EXISTS description;
ALTER TABLE products DROP COLUMN IF EXISTS brand;
