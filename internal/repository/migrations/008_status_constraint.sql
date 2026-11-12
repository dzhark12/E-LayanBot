ALTER TABLE permohonan DROP CONSTRAINT IF EXISTS permohonan_status_check;
ALTER TABLE permohonan ADD CONSTRAINT permohonan_status_check CHECK (status IN ('Menunggu', 'Dikerjakan', 'Perlu Perbaikan', 'Diperbaiki', 'Selesai', 'Ditolak', 'Dibatalkan'));
