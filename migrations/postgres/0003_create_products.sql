CREATE TABLE IF NOT EXISTS products (
    id              TEXT         PRIMARY KEY,
    name            VARCHAR(100) NOT NULL UNIQUE,
    aliases         TEXT[]       NOT NULL DEFAULT '{}',
    default_unit    VARCHAR(20)  NOT NULL DEFAULT '个',
    reference_price NUMERIC(12,2),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
