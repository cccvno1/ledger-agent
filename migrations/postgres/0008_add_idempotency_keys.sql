-- Idempotency key for safely retrying failed multi-row commits.
-- Nullable for back-compat. The unique index is unconditional because
-- Postgres treats NULL as distinct in unique indexes, but ON CONFLICT
-- requires a non-partial constraint to match conflict_target.
ALTER TABLE entries ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
DROP INDEX IF EXISTS uniq_entries_idempotency_key;
CREATE UNIQUE INDEX IF NOT EXISTS uniq_entries_idempotency_key
    ON entries(idempotency_key);

ALTER TABLE payments ADD COLUMN IF NOT EXISTS idempotency_key TEXT;
DROP INDEX IF EXISTS uniq_payments_idempotency_key;
CREATE UNIQUE INDEX IF NOT EXISTS uniq_payments_idempotency_key
    ON payments(idempotency_key);
