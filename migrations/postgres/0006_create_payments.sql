CREATE TABLE IF NOT EXISTS payments (
    id           TEXT           PRIMARY KEY,
    customer_id  TEXT           NOT NULL REFERENCES customers(id),
    amount       NUMERIC(14,2) NOT NULL,
    payment_date DATE           NOT NULL,
    notes        TEXT,
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payments_customer_id ON payments(customer_id);
