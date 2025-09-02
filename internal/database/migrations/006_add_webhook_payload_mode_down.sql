-- +goose Down
-- Remove campo webhook_payload_mode da tabela sessions

-- Remove índice
DROP INDEX IF EXISTS idx_sessions_webhook_payload_mode;

-- Remove coluna
ALTER TABLE sessions 
DROP COLUMN IF EXISTS webhook_payload_mode;