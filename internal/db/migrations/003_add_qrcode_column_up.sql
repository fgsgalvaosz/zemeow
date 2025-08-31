-- Migração 003: Adicionar coluna qrcode para armazenar QR codes temporários
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS qrcode TEXT;

-- Índice para performance na busca por QR code
CREATE INDEX IF NOT EXISTS idx_sessions_qrcode ON sessions(qrcode);
