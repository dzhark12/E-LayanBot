ALTER TABLE status_log ADD COLUMN IF NOT EXISTS bot_repair_request_id UUID;
CREATE UNIQUE INDEX IF NOT EXISTS status_log_bot_repair_request_id_uq
    ON status_log(bot_repair_request_id)
    WHERE bot_repair_request_id IS NOT NULL;
