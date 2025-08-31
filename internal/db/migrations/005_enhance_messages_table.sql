-- Migration: 005_enhance_messages_table.sql
-- Description: Enhance messages table to support all WhatsApp message types and improve relationships

-- Add new columns to messages table
ALTER TABLE messages 
ADD COLUMN IF NOT EXISTS whatsapp_message_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS recipient_jid VARCHAR(255),
ADD COLUMN IF NOT EXISTS raw_message JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS media_sha256 VARCHAR(64),
ADD COLUMN IF NOT EXISTS media_key BYTEA,
ADD COLUMN IF NOT EXISTS is_ephemeral BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS is_view_once BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS is_forwarded BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS is_edit BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS edit_version INTEGER DEFAULT 1,
ADD COLUMN IF NOT EXISTS reply_to_message_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS mentions JSONB DEFAULT '[]',
ADD COLUMN IF NOT EXISTS reaction_emoji VARCHAR(10),
ADD COLUMN IF NOT EXISTS reaction_timestamp TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS location_latitude DECIMAL(10,8),
ADD COLUMN IF NOT EXISTS location_longitude DECIMAL(11,8),
ADD COLUMN IF NOT EXISTS location_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS location_address TEXT,
ADD COLUMN IF NOT EXISTS contact_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS contact_phone VARCHAR(50),
ADD COLUMN IF NOT EXISTS contact_vcard TEXT,
ADD COLUMN IF NOT EXISTS sticker_pack_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS sticker_pack_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS group_invite_code VARCHAR(255),
ADD COLUMN IF NOT EXISTS group_invite_expiration TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS poll_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS poll_options JSONB DEFAULT '[]',
ADD COLUMN IF NOT EXISTS poll_selectable_count INTEGER DEFAULT 1,
ADD COLUMN IF NOT EXISTS error_message TEXT,
ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0;

-- Update existing columns with better constraints
ALTER TABLE messages 
ALTER COLUMN message_type SET DEFAULT 'text',
ALTER COLUMN direction SET DEFAULT 'incoming',
ALTER COLUMN status SET DEFAULT 'received';

-- Add new message types to check constraint
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_message_type_check;
ALTER TABLE messages ADD CONSTRAINT messages_message_type_check 
CHECK (message_type IN (
    'text', 'image', 'audio', 'video', 'document', 'sticker', 
    'location', 'contact', 'group_invite', 'poll', 'reaction',
    'system', 'call', 'unknown'
));

-- Add new status types to check constraint  
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_status_check;
ALTER TABLE messages ADD CONSTRAINT messages_status_check 
CHECK (status IN (
    'sent', 'delivered', 'read', 'failed', 'received', 
    'pending', 'server_ack', 'retry', 'undecryptable'
));

-- Create optimized indexes for message queries
CREATE INDEX IF NOT EXISTS idx_messages_whatsapp_message_id 
ON messages(session_id, whatsapp_message_id);

CREATE INDEX IF NOT EXISTS idx_messages_chat_sender 
ON messages(session_id, chat_jid, sender_jid);

CREATE INDEX IF NOT EXISTS idx_messages_recipient 
ON messages(session_id, recipient_jid) 
WHERE recipient_jid IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_reply_to 
ON messages(session_id, reply_to_message_id) 
WHERE reply_to_message_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_media_type 
ON messages(session_id, message_type, media_type) 
WHERE media_type IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_ephemeral 
ON messages(session_id, is_ephemeral, timestamp) 
WHERE is_ephemeral = TRUE;

CREATE INDEX IF NOT EXISTS idx_messages_reactions 
ON messages(session_id, chat_jid, reaction_emoji, reaction_timestamp) 
WHERE reaction_emoji IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_location 
ON messages(session_id, location_latitude, location_longitude) 
WHERE location_latitude IS NOT NULL AND location_longitude IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_messages_full_text_search 
ON messages USING gin(to_tsvector('portuguese', coalesce(content, '') || ' ' || coalesce(caption, '')));

-- Create partial indexes for performance
CREATE INDEX IF NOT EXISTS idx_messages_unread_incoming 
ON messages(session_id, chat_jid, timestamp DESC) 
WHERE direction = 'incoming' AND status != 'read';

CREATE INDEX IF NOT EXISTS idx_messages_failed 
ON messages(session_id, timestamp DESC) 
WHERE status = 'failed';

CREATE INDEX IF NOT EXISTS idx_messages_media_pending 
ON messages(session_id, timestamp DESC) 
WHERE message_type IN ('image', 'audio', 'video', 'document') 
AND media_url IS NULL;

-- Add foreign key constraints for better data integrity
-- Note: We'll add these after ensuring the related tables exist and have proper data

-- Create function to update message statistics in sessions table
CREATE OR REPLACE FUNCTION update_session_message_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- Update counters on insert
        IF NEW.direction = 'incoming' THEN
            UPDATE sessions 
            SET messages_received = messages_received + 1,
                last_activity = GREATEST(last_activity, NEW.timestamp)
            WHERE id = NEW.session_id;
        ELSE
            UPDATE sessions 
            SET messages_sent = messages_sent + 1,
                last_activity = GREATEST(last_activity, NEW.timestamp)
            WHERE id = NEW.session_id;
        END IF;
    ELSIF TG_OP = 'DELETE' THEN
        -- Update counters on delete
        IF OLD.direction = 'incoming' THEN
            UPDATE sessions 
            SET messages_received = GREATEST(0, messages_received - 1)
            WHERE id = OLD.session_id;
        ELSE
            UPDATE sessions 
            SET messages_sent = GREATEST(0, messages_sent - 1)
            WHERE id = OLD.session_id;
        END IF;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update session statistics
DROP TRIGGER IF EXISTS trigger_update_session_message_stats ON messages;
CREATE TRIGGER trigger_update_session_message_stats
    AFTER INSERT OR DELETE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_session_message_stats();

-- Create function to clean up old ephemeral messages
CREATE OR REPLACE FUNCTION cleanup_ephemeral_messages()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete ephemeral messages older than 7 days
    DELETE FROM messages 
    WHERE is_ephemeral = TRUE 
    AND timestamp < NOW() - INTERVAL '7 days';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON COLUMN messages.whatsapp_message_id IS 'Internal WhatsApp message ID';
COMMENT ON COLUMN messages.raw_message IS 'Complete WhatsApp message object as JSON';
COMMENT ON COLUMN messages.is_ephemeral IS 'True if message is ephemeral (disappearing)';
COMMENT ON COLUMN messages.is_view_once IS 'True if message is view once media';
COMMENT ON COLUMN messages.is_forwarded IS 'True if message was forwarded';
COMMENT ON COLUMN messages.mentions IS 'Array of mentioned JIDs in the message';
COMMENT ON COLUMN messages.reaction_emoji IS 'Emoji used for reaction to another message';
COMMENT ON COLUMN messages.reply_to_message_id IS 'ID of message this is replying to';
COMMENT ON COLUMN messages.retry_count IS 'Number of times message delivery was retried';
