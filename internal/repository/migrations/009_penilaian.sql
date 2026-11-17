CREATE TABLE IF NOT EXISTS penilaian (
    permohonan_id UUID PRIMARY KEY REFERENCES permohonan(id) ON DELETE CASCADE,
    nilai SMALLINT CHECK (nilai BETWEEN 1 AND 5),
    diminta_pada TIMESTAMPTZ NOT NULL DEFAULT now(),
    dinilai_pada TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS penilaian_pending_idx ON penilaian(diminta_pada) WHERE nilai IS NULL;