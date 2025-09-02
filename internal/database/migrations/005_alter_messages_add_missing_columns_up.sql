-- Migração 004: Adicionar colunas ausentes à tabela messages

-- Adicionar apenas as colunas essenciais que estão faltando
ALTER TABLE messages ADD COLUMN IF NOT EXISTS whatsapp_message_id VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS raw_message JSONB;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_key BYTEA;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_from_me BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS raw_message JSONB;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS media_key BYTEA;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS minio_media_id VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS minio_path VARCHAR(500);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS minio_url VARCHAR(500);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS minio_bucket VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS quoted_content TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reply_to_message_id VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS context_info JSONB;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_from_me BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_ephemeral BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_view_once BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_forwarded BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS is_edit BOOLEAN DEFAULT FALSE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS edit_version INTEGER DEFAULT 0;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS mentions TEXT[];
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reaction_emoji VARCHAR(10);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS reaction_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS location_latitude DOUBLE PRECISION;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS location_longitude DOUBLE PRECISION;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS location_name VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS location_address TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS contact_name VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS contact_phone VARCHAR(50);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS contact_vcard TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS sticker_pack_id VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS sticker_pack_name VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS group_invite_code VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS group_invite_expiration TIMESTAMP WITH TIME ZONE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS poll_name VARCHAR(255);
ALTER TABLE messages ADD COLUMN IF NOT EXISTS poll_options JSONB;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS poll_selectable_count INTEGER;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS error_message TEXT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0;

-- Renomear colunas se necessário (from_jid -> sender_jid, to_jid -> recipient_jid)
-- Nota: Isso pode falhar se as colunas já existirem com nomes diferentes
-- ALTER TABLE messages RENAME COLUMN from_jid TO sender_jid;
-- ALTER TABLE messages RENAME COLUMN to_jid TO recipient_jid;

-- Criar índices para as novas colunas
CREATE INDEX IF NOT EXISTS idx_messages_whatsapp_message_id ON messages(whatsapp_message_id) WHERE whatsapp_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_sender_jid ON messages(from_jid);
CREATE INDEX IF NOT EXISTS idx_messages_recipient_jid ON messages(to_jid) WHERE to_jid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_is_from_me ON messages(is_from_me);
CREATE INDEX IF NOT EXISTS idx_messages_media_type ON messages(media_type) WHERE media_type IS NOT NULL;