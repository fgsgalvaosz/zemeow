-- +goose Up
-- Adiciona campo webhook_payload_mode na tabela sessions
-- Este campo controla o modo de payload dos webhooks: 'processed', 'raw' ou 'both'

ALTER TABLE sessions 
ADD COLUMN webhook_payload_mode VARCHAR(10) DEFAULT 'processed' CHECK (webhook_payload_mode IN ('processed', 'raw', 'both'));

-- Comentário explicativo
COMMENT ON COLUMN sessions.webhook_payload_mode IS 'Modo de payload do webhook: processed (padrão), raw (dados brutos da whatsmeow) ou both (ambos os formatos)';

-- Índice para otimizar queries por payload mode
CREATE INDEX idx_sessions_webhook_payload_mode ON sessions(webhook_payload_mode);