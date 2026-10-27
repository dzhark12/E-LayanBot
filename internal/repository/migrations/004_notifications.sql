ALTER TABLE notifikasi_log ADD COLUMN IF NOT EXISTS sent_at TIMESTAMPTZ;
ALTER TABLE notifikasi_log ADD COLUMN IF NOT EXISTS attempt_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE notifikasi_log ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE notifikasi_log ADD COLUMN IF NOT EXISTS claimed_until TIMESTAMPTZ;
CREATE UNIQUE INDEX IF NOT EXISTS notifikasi_log_request_status_uq ON notifikasi_log(permohonan_id,status_terkirim);
CREATE INDEX IF NOT EXISTS notifikasi_log_pending_idx ON notifikasi_log(next_attempt_at,waktu) WHERE sent_at IS NULL;
