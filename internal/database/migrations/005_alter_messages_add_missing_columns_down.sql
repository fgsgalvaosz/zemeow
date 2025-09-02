-- Migração 004: Reverter adição de colunas à tabela messages

-- Remover índices das novas colunas
DROP INDEX IF EXISTS idx_messages_whatsapp_message_id;
DROP INDEX IF EXISTS idx_messages_sender_jid;
DROP INDEX IF EXISTS idx_messages_recipient_jid;
DROP INDEX IF EXISTS idx_messages_is_from_me;
DROP INDEX IF EXISTS idx_messages_media_type;

-- Remover colunas adicionadas
ALTER TABLE messages DROP COLUMN IF EXISTS whatsapp_message_id;
ALTER TABLE messages DROP COLUMN IF EXISTS raw_message;
ALTER TABLE messages DROP COLUMN IF EXISTS media_key;
ALTER TABLE messages DROP COLUMN IF EXISTS is_from_me;
ALTER TABLE messages DROP COLUMN IF EXISTS raw_message;
ALTER TABLE messages DROP COLUMN IF EXISTS media_key;
ALTER TABLE messages DROP COLUMN IF EXISTS minio_media_id;
ALTER TABLE messages DROP COLUMN IF EXISTS minio_path;
ALTER TABLE messages DROP COLUMN IF EXISTS minio_url;
ALTER TABLE messages DROP COLUMN IF EXISTS minio_bucket;
ALTER TABLE messages DROP COLUMN IF EXISTS quoted_content;
ALTER TABLE messages DROP COLUMN IF EXISTS reply_to_message_id;
ALTER TABLE messages DROP COLUMN IF EXISTS context_info;
ALTER TABLE messages DROP COLUMN IF EXISTS is_from_me;
ALTER TABLE messages DROP COLUMN IF EXISTS is_ephemeral;
ALTER TABLE messages DROP COLUMN IF EXISTS is_view_once;
ALTER TABLE messages DROP COLUMN IF EXISTS is_forwarded;
ALTER TABLE messages DROP COLUMN IF EXISTS is_edit;
ALTER TABLE messages DROP COLUMN IF EXISTS edit_version;
ALTER TABLE messages DROP COLUMN IF EXISTS mentions;
ALTER TABLE messages DROP COLUMN IF EXISTS reaction_emoji;
ALTER TABLE messages DROP COLUMN IF EXISTS reaction_timestamp;
ALTER TABLE messages DROP COLUMN IF EXISTS location_latitude;
ALTER TABLE messages DROP COLUMN IF EXISTS location_longitude;
ALTER TABLE messages DROP COLUMN IF EXISTS location_name;
ALTER TABLE messages DROP COLUMN IF EXISTS location_address;
ALTER TABLE messages DROP COLUMN IF EXISTS contact_name;
ALTER TABLE messages DROP COLUMN IF EXISTS contact_phone;
ALTER TABLE messages DROP COLUMN IF EXISTS contact_vcard;
ALTER TABLE messages DROP COLUMN IF EXISTS sticker_pack_id;
ALTER TABLE messages DROP COLUMN IF EXISTS sticker_pack_name;
ALTER TABLE messages DROP COLUMN IF EXISTS group_invite_code;
ALTER TABLE messages DROP COLUMN IF EXISTS group_invite_expiration;
ALTER TABLE messages DROP COLUMN IF EXISTS poll_name;
ALTER TABLE messages DROP COLUMN IF EXISTS poll_options;
ALTER TABLE messages DROP COLUMN IF EXISTS poll_selectable_count;
ALTER TABLE messages DROP COLUMN IF EXISTS error_message;
ALTER TABLE messages DROP COLUMN IF EXISTS retry_count;