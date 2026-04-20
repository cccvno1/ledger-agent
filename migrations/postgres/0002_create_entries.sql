CREATE TABLE IF NOT EXISTS entries (
    id            TEXT         PRIMARY KEY,
    customer_id   TEXT         NOT NULL REFERENCES customers(id),
    customer_name VARCHAR(100) NOT NULL,
    product_name  VARCHAR(100) NOT NULL,
    unit_price    NUMERIC(12,2) NOT NULL,
    quantity      NUMERIC(12,3) NOT NULL,
    amount        NUMERIC(14,2) NOT NULL,
    entry_date    DATE         NOT NULL,
    is_settled    BOOLEAN      NOT NULL DEFAULT FALSE,
    settled_at    TIMESTAMPTZ,
    notes         TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_entries_customer_id   ON entries(customer_id);
CREATE INDEX IF NOT EXISTS idx_entries_entry_date    ON entries(entry_date);
CREATE INDEX IF NOT EXISTS idx_entries_is_settled    ON entries(is_settled);
