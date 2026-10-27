ALTER TABLE permohonan ADD COLUMN IF NOT EXISTS bot_request_id UUID;
CREATE UNIQUE INDEX IF NOT EXISTS permohonan_bot_request_id_uq ON permohonan(bot_request_id) WHERE bot_request_id IS NOT NULL;
