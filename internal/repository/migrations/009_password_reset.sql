-- 1. Tambah kolom email pada admin_user jika belum ada
ALTER TABLE admin_user ADD COLUMN IF NOT EXISTS email TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS admin_user_email_uq ON admin_user (LOWER(email)) WHERE email IS NOT NULL AND email <> '';

-- 2. Tabel password_reset_token
CREATE TABLE IF NOT EXISTS password_reset_token (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    admin_id BIGINT NOT NULL REFERENCES admin_user(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS password_reset_token_admin_idx ON password_reset_token(admin_id);
CREATE INDEX IF NOT EXISTS password_reset_token_expiry_idx ON password_reset_token(expires_at) WHERE used_at IS NULL;

-- 3. Tabel password_reset_attempt untuk rate limiting IP & target email
CREATE TABLE IF NOT EXISTS password_reset_attempt (
    target_hash BYTEA NOT NULL,
    attempted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS password_reset_attempt_idx ON password_reset_attempt(target_hash, attempted_at);

-- 4. Tabel audit log untuk aktivitas administrator (termasuk reset password oleh admin)
CREATE TABLE IF NOT EXISTS admin_audit_log (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    admin_id BIGINT REFERENCES admin_user(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    target_user_id BIGINT REFERENCES admin_user(id) ON DELETE SET NULL,
    details TEXT NOT NULL DEFAULT '',
    ip_address TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS admin_audit_log_created_idx ON admin_audit_log(created_at DESC);

