-- Migração 004 DOWN: Remover foreign key corrigida

DO $$
BEGIN
    -- Remover foreign key sessions -> whatsmeow_device
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_sessions_whatsmeow_device') THEN
        ALTER TABLE sessions DROP CONSTRAINT fk_sessions_whatsmeow_device;
        RAISE NOTICE 'Foreign key constraint sessions -> whatsmeow_device removed';
    END IF;
END;
$$ LANGUAGE plpgsql;
