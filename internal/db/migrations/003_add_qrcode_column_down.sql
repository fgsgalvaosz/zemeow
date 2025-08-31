-- Migração 003 (rollback): Remover coluna qrcode
DROP INDEX IF EXISTS idx_sessions_qrcode;
ALTER TABLE sessions DROP COLUMN IF EXISTS qrcode;
