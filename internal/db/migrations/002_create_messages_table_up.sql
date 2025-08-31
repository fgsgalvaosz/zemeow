-- Migração 002: Criar tabela messages completa
-- Versão unificada para v1.0.0

CREATE TABLE IF NOT EXISTS messages (
    -- Identificadores
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id VARCHAR(255) UNIQUE NOT NULL, -- WhatsApp message ID
    session_id UUID NOT NULL,
    
    -- Participantes
    from_jid VARCHAR(255) NOT NULL, -- Remetente
    to_jid VARCHAR(255) NOT NULL,   -- Destinatário
    chat_jid VARCHAR(255) NOT NULL, -- Chat (pode ser grupo)
    
    -- Conteúdo da mensagem
    message_type VARCHAR(50) NOT NULL, -- text, image, audio, video, document, etc.
    content TEXT, -- Conteúdo textual
    caption TEXT, -- Legenda para mídias
    
    -- Mídia
    media_type VARCHAR(50), -- image/jpeg, audio/ogg, etc.
    media_size BIGINT, -- Tamanho em bytes
    media_filename VARCHAR(255), -- Nome original do arquivo
    media_mimetype VARCHAR(100), -- MIME type
    media_sha256 VARCHAR(64), -- Hash SHA256 da mídia
    media_url VARCHAR(500), -- URL da mídia (MinIO)
    media_path VARCHAR(500), -- Caminho no storage
    
    -- MinIO específico
    minio_bucket VARCHAR(255), -- Bucket do MinIO
    minio_object_key VARCHAR(500), -- Chave do objeto no MinIO
    minio_public_url VARCHAR(500), -- URL pública do MinIO
    minio_content_type VARCHAR(100), -- Content-Type no MinIO
    minio_etag VARCHAR(255), -- ETag do MinIO
    
    -- Status e metadados
    direction VARCHAR(20) NOT NULL, -- incoming, outgoing
    status VARCHAR(50) DEFAULT 'received', -- received, sent, delivered, read, error
    is_forwarded BOOLEAN DEFAULT FALSE,
    is_broadcast BOOLEAN DEFAULT FALSE,
    is_group BOOLEAN DEFAULT FALSE,
    
    -- Timestamps
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL, -- Timestamp da mensagem
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Dados brutos e metadados
    raw_data JSONB, -- Dados brutos do WhatsApp
    metadata JSONB DEFAULT '{}', -- Metadados adicionais
    
    -- Relacionamentos
    reply_to_message_id VARCHAR(255), -- ID da mensagem respondida
    quoted_message_id VARCHAR(255), -- ID da mensagem citada
    
    -- Constraints
    CONSTRAINT fk_messages_session FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    CONSTRAINT check_direction_valid CHECK (direction IN ('incoming', 'outgoing')),
    CONSTRAINT check_status_valid CHECK (status IN ('received', 'sent', 'delivered', 'read', 'error', 'pending')),
    CONSTRAINT check_message_type_valid CHECK (message_type IN (
        'text', 'image', 'audio', 'video', 'document', 'sticker', 
        'location', 'contact', 'reaction', 'system', 'unknown'
    ))
);

-- Índices para performance
CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_message_id ON messages(message_id);
CREATE INDEX IF NOT EXISTS idx_messages_chat_jid ON messages(chat_jid);
CREATE INDEX IF NOT EXISTS idx_messages_from_jid ON messages(from_jid);
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_messages_direction ON messages(direction);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_message_type ON messages(message_type);
CREATE INDEX IF NOT EXISTS idx_messages_media_type ON messages(media_type) WHERE media_type IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

-- Índices compostos para queries comuns
CREATE INDEX IF NOT EXISTS idx_messages_session_chat_timestamp ON messages(session_id, chat_jid, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_session_direction_timestamp ON messages(session_id, direction, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_session_status ON messages(session_id, status);

-- Trigger para atualizar updated_at
CREATE TRIGGER update_messages_updated_at 
    BEFORE UPDATE ON messages 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
