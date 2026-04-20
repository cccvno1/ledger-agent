CREATE TABLE IF NOT EXISTS customers (
    id         TEXT        PRIMARY KEY,
    name       VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
