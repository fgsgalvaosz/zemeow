-- Migração 001: Criar tabela sessions e estrutura básica
-- Criar extensões necessárias (PostgreSQL)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Criar tabela sessions (usando UUID como identificador principal e dual mode)
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    api_key VARCHAR(255) UNIQUE NOT NULL,
    jid VARCHAR(255),
    status VARCHAR(50) DEFAULT 'disconnected',
    proxy_enabled BOOLEAN DEFAULT FALSE,
    proxy_host VARCHAR(255),
    proxy_port INTEGER,
    proxy_username VARCHAR(255),
    proxy_password VARCHAR(255),
    webhook_url VARCHAR(500),
    webhook_events TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_connected_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    messages_received INTEGER DEFAULT 0,
    messages_sent INTEGER DEFAULT 0,
    reconnections INTEGER DEFAULT 0,
    last_activity TIMESTAMP WITH TIME ZONE,
    
    -- Constraint para garantir que name seja URL-safe (dual mode)
    CONSTRAINT check_name_url_safe CHECK (name ~ '^[a-zA-Z0-9_-]+$' AND length(name) >= 3 AND length(name) <= 50)
);

-- Índices para performance (dual mode: UUID e name)
CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_name_unique ON sessions(name);
CREATE INDEX IF NOT EXISTS idx_sessions_name_lookup ON sessions(name);
CREATE INDEX IF NOT EXISTS idx_sessions_api_key ON sessions(api_key);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_jid ON sessions(jid);
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at);

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
