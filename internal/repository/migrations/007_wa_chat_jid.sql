ALTER TABLE permohonan ADD COLUMN IF NOT EXISTS wa_chat_jid TEXT;
CREATE INDEX IF NOT EXISTS permohonan_wa_jid_idx ON permohonan(wa_jid);
