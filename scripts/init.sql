-- Inicialização do banco de dados ZeMeow
-- Este script é executado automaticamente quando o container PostgreSQL é criado

-- Criar extensões necessárias
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Criar schema para organização
CREATE SCHEMA IF NOT EXISTS whatsmeow;

-- Tabela para armazenar informações das sessões
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    jid VARCHAR(255),
    status VARCHAR(50) DEFAULT 'disconnected',
    proxy_enabled BOOLEAN DEFAULT FALSE,
    proxy_host VARCHAR(255),
    proxy_port INTEGER,
    proxy_username VARCHAR(255),
    proxy_password VARCHAR(255),
    webhook_url VARCHAR(500),
    webhook_events TEXT[], -- Array de eventos para webhook
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_connected_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'
);

-- Índices para performance
CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
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
$$ language 'plpgsql';

-- Trigger para atualizar updated_at
CREATE TRIGGER update_sessions_updated_at 
    BEFORE UPDATE ON sessions 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Inserir dados de exemplo (opcional, apenas para desenvolvimento)
INSERT INTO sessions (session_id, name, token, status) 
VALUES 
    ('demo-session-1', 'Sessão Demo 1', 'demo_token_1', 'disconnected'),
    ('demo-session-2', 'Sessão Demo 2', 'demo_token_2', 'disconnected')
ON CONFLICT (session_id) DO NOTHING;

-- Comentários para documentação
COMMENT ON TABLE sessions IS 'Tabela para armazenar informações das sessões WhatsApp';
COMMENT ON COLUMN sessions.session_id IS 'Identificador único da sessão';
COMMENT ON COLUMN sessions.token IS 'Token de autenticação da sessão';
COMMENT ON COLUMN sessions.jid IS 'JID do WhatsApp (WhatsApp ID)';
COMMENT ON COLUMN sessions.status IS 'Status da conexão: disconnected, connecting, connected, authenticated';
COMMENT ON COLUMN sessions.webhook_events IS 'Array de eventos que devem ser enviados para o webhook';
COMMENT ON COLUMN sessions.metadata IS 'Dados adicionais em formato JSON';

-- Criar usuário específico para a aplicação (se necessário)
-- Este comando pode falhar se o usuário já existir, por isso usamos DO $$ block
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'zemeow_app') THEN
        CREATE ROLE zemeow_app WITH LOGIN PASSWORD 'app_password_change_in_production';
    END IF;
END
$$;

-- Conceder permissões necessárias
GRANT USAGE ON SCHEMA public TO zemeow_app;
GRANT USAGE ON SCHEMA whatsmeow TO zemeow_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO zemeow_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA whatsmeow TO zemeow_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO zemeow_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA whatsmeow TO zemeow_app;

-- Configurações de performance
ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements';
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;

-- Recarregar configurações
SELECT pg_reload_conf();

-- Log de inicialização
DO $$
BEGIN
    RAISE NOTICE 'ZeMeow database initialized successfully at %', NOW();
END
$$;