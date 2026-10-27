CREATE SEQUENCE IF NOT EXISTS permohonan_code_seq MINVALUE 1 MAXVALUE 99999 NO CYCLE;

CREATE TABLE IF NOT EXISTS admin_user (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'operator')),
    aktif BOOLEAN NOT NULL DEFAULT TRUE,
    dibuat_pada TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS admin_session (
    token_hash BYTEA PRIMARY KEY,
    csrf_token TEXT NOT NULL,
    admin_id BIGINT NOT NULL REFERENCES admin_user(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    dibuat_pada TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS admin_session_expiry_idx ON admin_session(expires_at);

CREATE TABLE IF NOT EXISTS admin_login_attempt (
    ip_hash BYTEA NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS admin_login_attempt_idx ON admin_login_attempt(ip_hash, attempted_at);

CREATE TABLE IF NOT EXISTS jenis_surat (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kode VARCHAR(2) NOT NULL UNIQUE CHECK (kode ~ '^[A-Z]{2}$'),
    nama TEXT NOT NULL,
    syarat TEXT[] NOT NULL DEFAULT '{}',
    aktif BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS permohonan (
    id UUID PRIMARY KEY,
    kode_unik VARCHAR(7) NOT NULL UNIQUE CHECK (kode_unik ~ '^[A-Z]{2}[0-9]{5}$'),
    jenis_id BIGINT NOT NULL REFERENCES jenis_surat(id),
    wa_jid TEXT NOT NULL,
    wa_number_hash BYTEA NOT NULL,
    ringkasan TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'Menunggu'
        CHECK (status IN ('Menunggu', 'Dikerjakan', 'Perlu Perbaikan', 'Selesai', 'Ditolak')),
    data_identitas JSONB NOT NULL DEFAULT '{}'::jsonb,
    consent_version TEXT NOT NULL,
    consented_at TIMESTAMPTZ NOT NULL,
    dibuat_pada TIMESTAMPTZ NOT NULL DEFAULT now(),
    diubah_pada TIMESTAMPTZ NOT NULL DEFAULT now(),
    selesai_pada TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS permohonan_status_time_idx ON permohonan(status, dibuat_pada DESC);

CREATE TABLE IF NOT EXISTS lampiran (
    id UUID PRIMARY KEY,
    permohonan_id UUID NOT NULL REFERENCES permohonan(id) ON DELETE CASCADE,
    jenis TEXT NOT NULL CHECK (jenis IN ('KTP', 'KK', 'RT', 'LAINNYA')),
    storage_key TEXT NOT NULL UNIQUE,
    mime TEXT NOT NULL CHECK (mime IN ('image/jpeg', 'image/png')),
    ukuran BIGINT NOT NULL CHECK (ukuran > 0),
    dibuat_pada TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS lampiran_permohonan_idx ON lampiran(permohonan_id);

CREATE TABLE IF NOT EXISTS status_log (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    permohonan_id UUID NOT NULL REFERENCES permohonan(id) ON DELETE CASCADE,
    status_lama TEXT NOT NULL,
    status_baru TEXT NOT NULL,
    admin_id BIGINT REFERENCES admin_user(id),
    aktor TEXT NOT NULL DEFAULT 'system',
    waktu TIMESTAMPTZ NOT NULL DEFAULT now(),
    catatan TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS status_log_permohonan_idx ON status_log(permohonan_id, waktu DESC);

CREATE TABLE IF NOT EXISTS status_lookup_attempt (
    wa_hash BYTEA NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS status_lookup_attempt_idx ON status_lookup_attempt(wa_hash, attempted_at);

CREATE TABLE IF NOT EXISTS faq (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pertanyaan TEXT NOT NULL,
    kata_kunci TEXT[] NOT NULL DEFAULT '{}',
    jawaban TEXT NOT NULL,
    kategori TEXT NOT NULL DEFAULT '',
    aktif BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS notifikasi_log (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    permohonan_id UUID NOT NULL REFERENCES permohonan(id) ON DELETE CASCADE,
    status_terkirim TEXT NOT NULL,
    waktu TIMESTAMPTZ NOT NULL DEFAULT now(),
    error TEXT NOT NULL DEFAULT ''
);
