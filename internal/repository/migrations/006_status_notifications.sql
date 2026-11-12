ALTER TABLE notifikasi_log ADD COLUMN IF NOT EXISTS status_log_id BIGINT REFERENCES status_log(id) ON DELETE CASCADE;
DROP INDEX IF EXISTS notifikasi_log_request_status_uq;
CREATE UNIQUE INDEX IF NOT EXISTS notifikasi_log_status_log_uq ON notifikasi_log(status_log_id) WHERE status_log_id IS NOT NULL;

ALTER TABLE permohonan DROP CONSTRAINT IF EXISTS permohonan_status_check;
ALTER TABLE permohonan ADD CONSTRAINT permohonan_status_check
    CHECK (status IN ('Menunggu', 'Dikerjakan', 'Perlu Perbaikan', 'Diperbaiki', 'Selesai', 'Ditolak', 'Dibatalkan'));
