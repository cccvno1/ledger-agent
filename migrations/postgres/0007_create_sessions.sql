CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT        PRIMARY KEY,
    data       JSONB       NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
