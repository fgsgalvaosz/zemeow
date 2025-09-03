-- +goose Down
-- Reverter criação do schema inicial

-- Remover triggers
DROP TRIGGER IF EXISTS update_messages_updated_at ON messages;
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;

-- Remover função
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Remover índices de messages
DROP INDEX IF EXISTS idx_messages_session_from_timestamp;
DROP INDEX IF EXISTS idx_messages_session_status;
DROP INDEX IF EXISTS idx_messages_session_direction_timestamp;
DROP INDEX IF EXISTS idx_messages_session_chat_timestamp;
DROP INDEX IF EXISTS idx_messages_updated_at;
DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_messages_is_from_me;
DROP INDEX IF EXISTS idx_messages_media_type;
DROP INDEX IF EXISTS idx_messages_message_type;
DROP INDEX IF EXISTS idx_messages_status;
DROP INDEX IF EXISTS idx_messages_direction;
DROP INDEX IF EXISTS idx_messages_timestamp;
DROP INDEX IF EXISTS idx_messages_to_jid;
DROP INDEX IF EXISTS idx_messages_from_jid;
DROP INDEX IF EXISTS idx_messages_chat_jid;
DROP INDEX IF EXISTS idx_messages_whatsapp_message_id;
DROP INDEX IF EXISTS idx_messages_message_id;
DROP INDEX IF EXISTS idx_messages_session_id;

-- Remover índices de sessions
DROP INDEX IF EXISTS idx_sessions_proxy_enabled;
DROP INDEX IF EXISTS idx_sessions_webhook_url;
DROP INDEX IF EXISTS idx_sessions_last_activity;
DROP INDEX IF EXISTS idx_sessions_created_at;
DROP INDEX IF EXISTS idx_sessions_status;
DROP INDEX IF EXISTS idx_sessions_jid;
DROP INDEX IF EXISTS idx_sessions_api_key;
DROP INDEX IF EXISTS idx_sessions_name;

-- Remover tabelas
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS sessions;