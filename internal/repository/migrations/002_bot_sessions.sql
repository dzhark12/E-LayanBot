CREATE TABLE IF NOT EXISTS sesi_bot (
    jid TEXT PRIMARY KEY,
    flow TEXT NOT NULL DEFAULT '',
    step TEXT NOT NULL DEFAULT '',
    data JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS sesi_bot_expiry_idx ON sesi_bot(updated_at);
