-- Rollback da migração 001: Remover tabela sessions e estrutura

-- Remover trigger
DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;

-- Remover função
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Remover índices
DROP INDEX IF EXISTS idx_sessions_created_at;
DROP INDEX IF EXISTS idx_sessions_jid;
DROP INDEX IF EXISTS idx_sessions_status;
DROP INDEX IF EXISTS idx_sessions_api_key;
DROP INDEX IF EXISTS idx_sessions_name_lookup;
DROP INDEX IF EXISTS idx_sessions_name_unique;

-- Remover tabela
DROP TABLE IF EXISTS sessions;

-- Remover extensões (cuidado: outras tabelas podem usar)
-- DROP EXTENSION IF EXISTS "pgcrypto";
-- DROP EXTENSION IF EXISTS "uuid-ossp";
