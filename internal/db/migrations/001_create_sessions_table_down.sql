-- Migração 001: Reverter criação da tabela sessions

-- Remover trigger
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;

-- Remover função
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Remover índices
DROP INDEX IF EXISTS idx_sessions_proxy_enabled;
DROP INDEX IF EXISTS idx_sessions_webhook_url;
DROP INDEX IF EXISTS idx_sessions_last_activity;
DROP INDEX IF EXISTS idx_sessions_created_at;
DROP INDEX IF EXISTS idx_sessions_status;
DROP INDEX IF EXISTS idx_sessions_jid;
DROP INDEX IF EXISTS idx_sessions_api_key;
DROP INDEX IF EXISTS idx_sessions_name;

-- Remover tabela
DROP TABLE IF EXISTS sessions;
