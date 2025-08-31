-- Migração 001: Criar tabela sessions completa
-- Versão unificada para v1.0.0

-- Criar extensões necessárias
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Criar tabela sessions com todos os campos necessários
CREATE TABLE IF NOT EXISTS sessions (
    -- Identificadores
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    api_key VARCHAR(255) UNIQUE NOT NULL,
    jid VARCHAR(255) UNIQUE, -- WhatsApp JID (único para foreign keys)
    
    -- Status e configuração
    status VARCHAR(50) DEFAULT 'disconnected',
    qr_code TEXT, -- QR Code para conexão
    
    -- Proxy
    proxy_enabled BOOLEAN DEFAULT FALSE,
    proxy_host VARCHAR(255),
    proxy_port INTEGER,
    proxy_username VARCHAR(255),
    proxy_password VARCHAR(255),
    
    -- Webhook
    webhook_url VARCHAR(500),
    webhook_events TEXT[],
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_connected_at TIMESTAMP WITH TIME ZONE,
    last_activity TIMESTAMP WITH TIME ZONE,
    
    -- Metadados e estatísticas
    metadata JSONB DEFAULT '{}',
    messages_received INTEGER DEFAULT 0,
    messages_sent INTEGER DEFAULT 0,
    reconnections INTEGER DEFAULT 0,
    
    -- MinIO fields para armazenamento de mídias
    s3_enabled BOOLEAN DEFAULT TRUE,
    s3_endpoint VARCHAR(255),
    s3_access_key VARCHAR(255),
    s3_secret_key VARCHAR(255),
    s3_bucket_name VARCHAR(255) DEFAULT 'zemeow-media',
    s3_use_ssl BOOLEAN DEFAULT FALSE,
    s3_region VARCHAR(100) DEFAULT 'us-east-1',
    s3_public_url VARCHAR(500),
    
    -- Constraints
    CONSTRAINT check_name_url_safe CHECK (name ~ '^[a-zA-Z0-9_-]+$' AND length(name) >= 3 AND length(name) <= 50),
    CONSTRAINT check_status_valid CHECK (status IN ('disconnected', 'connecting', 'connected', 'error', 'qr_code')),
    CONSTRAINT check_proxy_port CHECK (proxy_port IS NULL OR (proxy_port > 0 AND proxy_port <= 65535))
);

-- Índices para performance
CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_name ON sessions(name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_api_key ON sessions(api_key);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_jid ON sessions(jid) WHERE jid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at);
CREATE INDEX IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity) WHERE last_activity IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_webhook_url ON sessions(webhook_url) WHERE webhook_url IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_proxy_enabled ON sessions(proxy_enabled) WHERE proxy_enabled = true;

-- Função para atualizar updated_at automaticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger para atualizar updated_at automaticamente
CREATE TRIGGER update_sessions_updated_at 
    BEFORE UPDATE ON sessions 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
