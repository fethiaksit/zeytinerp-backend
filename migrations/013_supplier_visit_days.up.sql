ALTER TABLE suppliers
ADD COLUMN IF NOT EXISTS visit_days JSONB NOT NULL DEFAULT '[]'::jsonb;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'suppliers_visit_days_array_check'
          AND conrelid = 'suppliers'::regclass
    ) THEN
        ALTER TABLE suppliers
        ADD CONSTRAINT suppliers_visit_days_array_check
        CHECK (
            jsonb_typeof(visit_days) = 'array'
            AND visit_days <@ '["monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"]'::jsonb
        );
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_suppliers_visit_days
ON suppliers USING GIN (visit_days);
