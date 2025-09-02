-- Migração 002: Criar tabela messages completa
-- Versão unificada para v1.0.0

-- Drop table if it exists with old schema
DROP TABLE IF EXISTS messages;

CREATE TABLE messages (
    -- Identificadores
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id VARCHAR(255) UNIQUE NOT NULL, -- Internal message ID
    whatsapp_message_id VARCHAR(255), -- WhatsApp message ID
    session_id UUID NOT NULL,

    -- Participantes
    chat_jid VARCHAR(255) NOT NULL, -- Chat (pode ser grupo)
    from_jid VARCHAR(255) NOT NULL, -- Remetente
    to_jid VARCHAR(255), -- Destinatário (opcional para grupos)

    -- Conteúdo da mensagem
    message_type VARCHAR(50) NOT NULL, -- text, image, audio, video, document, etc.
    content TEXT, -- Conteúdo textual
    raw_message JSONB, -- Dados brutos do WhatsApp

    -- Mídia
    media_url VARCHAR(500), -- URL da mídia
    media_type VARCHAR(50), -- image/jpeg, audio/ogg, etc.
    media_size BIGINT, -- Tamanho em bytes
    media_filename VARCHAR(255), -- Nome original do arquivo
    media_sha256 VARCHAR(64), -- Hash SHA256 da mídia
    media_key BYTEA, -- Chave de criptografia da mídia

    -- MinIO
    minio_media_id VARCHAR(255), -- ID da mídia no MinIO
    minio_path VARCHAR(500), -- Caminho no MinIO
    minio_url VARCHAR(500), -- URL pública do MinIO
    minio_bucket VARCHAR(255), -- Bucket do MinIO

    -- Conteúdo adicional
    caption TEXT, -- Legenda para mídias
    quoted_message_id VARCHAR(255), -- ID da mensagem citada
    quoted_content TEXT, -- Conteúdo da mensagem citada
    reply_to_message_id VARCHAR(255), -- ID da mensagem respondida
    context_info JSONB, -- Informações de contexto

    -- Status
    direction VARCHAR(20) NOT NULL, -- incoming, outgoing
    status VARCHAR(50) DEFAULT 'received', -- received, sent, delivered, read, error, pending, server_ack, retry, undecryptable
    is_from_me BOOLEAN DEFAULT FALSE,
    is_ephemeral BOOLEAN DEFAULT FALSE,
    is_view_once BOOLEAN DEFAULT FALSE,
    is_forwarded BOOLEAN DEFAULT FALSE,
    is_edit BOOLEAN DEFAULT FALSE,
    edit_version INTEGER DEFAULT 0,

    -- Reações e menções
    mentions TEXT[], -- Lista de JIDs mencionados
    reaction_emoji VARCHAR(10), -- Emoji da reação
    reaction_timestamp TIMESTAMP WITH TIME ZONE, -- Timestamp da reação

    -- Localização
    location_latitude DOUBLE PRECISION,
    location_longitude DOUBLE PRECISION,
    location_name VARCHAR(255),
    location_address TEXT,

    -- Contato
    contact_name VARCHAR(255),
    contact_phone VARCHAR(50),
    contact_vcard TEXT,

    -- Sticker
    sticker_pack_id VARCHAR(255),
    sticker_pack_name VARCHAR(255),

    -- Grupo
    group_invite_code VARCHAR(255),
    group_invite_expiration TIMESTAMP WITH TIME ZONE,

    -- Enquete
    poll_name VARCHAR(255),
    poll_options JSONB,
    poll_selectable_count INTEGER,

    -- Erro e retry
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,

    -- Timestamps
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL, -- Timestamp da mensagem
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT fk_messages_session FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    CONSTRAINT check_direction_valid CHECK (direction IN ('incoming', 'outgoing')),
    CONSTRAINT check_status_valid CHECK (status IN ('received', 'sent', 'delivered', 'read', 'failed', 'pending', 'server_ack', 'retry', 'undecryptable')),
    CONSTRAINT check_message_type_valid CHECK (message_type IN (
        'text', 'image', 'audio', 'video', 'document', 'sticker',
        'location', 'contact', 'group_invite', 'poll', 'reaction', 'system', 'call', 'unknown'
    ))
);

-- Índices para performance
CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_whatsapp_message_id ON messages(whatsapp_message_id) WHERE whatsapp_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_chat_jid ON messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_messages_from_jid ON messages(from_jid);
CREATE INDEX IF NOT EXISTS idx_messages_to_jid ON messages(to_jid) WHERE to_jid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_messages_direction ON messages(direction);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_message_type ON messages(message_type);
CREATE INDEX IF NOT EXISTS idx_messages_media_type ON messages(media_type) WHERE media_type IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_is_from_me ON messages(is_from_me);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_updated_at ON messages(updated_at);

-- Índices compostos para queries comuns
CREATE INDEX IF NOT EXISTS idx_messages_session_chat_timestamp ON messages(session_id, chat_jid, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_session_direction_timestamp ON messages(session_id, direction, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_session_status ON messages(session_id, status);
CREATE INDEX IF NOT EXISTS idx_messages_session_from_timestamp ON messages(session_id, from_jid, timestamp DESC);

-- Trigger para atualizar updated_at
CREATE TRIGGER update_messages_updated_at 
    BEFORE UPDATE ON messages 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
