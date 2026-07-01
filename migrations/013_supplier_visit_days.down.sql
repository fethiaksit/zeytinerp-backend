DROP INDEX IF EXISTS idx_suppliers_visit_days;

ALTER TABLE suppliers
DROP CONSTRAINT IF EXISTS suppliers_visit_days_array_check;

ALTER TABLE suppliers
DROP COLUMN IF EXISTS visit_days;
