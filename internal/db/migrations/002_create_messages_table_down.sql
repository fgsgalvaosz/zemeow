-- Migração 002: Reverter criação da tabela messages

-- Remover trigger
DROP TRIGGER IF EXISTS update_messages_updated_at ON messages;

-- Remover índices
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

-- Nota: Tabela já foi removida na migração up para recriar com schema correto
-- DROP TABLE IF EXISTS messages;
