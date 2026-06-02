CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT,
    password_hash TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS username TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'users'
          AND column_name = 'email'
    ) THEN
        EXECUTE 'UPDATE users SET username = email WHERE (username IS NULL OR username = '''') AND email IS NOT NULL AND email <> ''''';
    END IF;
END $$;

UPDATE users
SET username = 'user_' || id
WHERE username IS NULL OR username = '';

UPDATE users
SET role = 'user'
WHERE role IS NULL OR role NOT IN ('admin', 'user');

ALTER TABLE users ALTER COLUMN username SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_role_check'
    ) THEN
        ALTER TABLE users
        ADD CONSTRAINT users_role_check
        CHECK (role IN ('admin', 'user'));
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM users WHERE username = 'admin') THEN
        IF EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_name = 'users'
              AND column_name = 'email'
        ) THEN
            EXECUTE 'INSERT INTO users (email, username, password_hash, name, role)
                     VALUES (''admin'', ''admin'', ''$2a$10$5cRRjhi9.sUZq2AXj76O7udKG7t9yBw4msnj6GtYkrNizmWP/7zyO'', ''Yönetici'', ''admin'')';
        ELSE
            INSERT INTO users (username, password_hash, name, role)
            VALUES (
                'admin',
                '$2a$10$5cRRjhi9.sUZq2AXj76O7udKG7t9yBw4msnj6GtYkrNizmWP/7zyO',
                'Yönetici',
                'admin'
            );
        END IF;
    END IF;
END $$;
