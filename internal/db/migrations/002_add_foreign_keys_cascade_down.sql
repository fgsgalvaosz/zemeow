-- Rollback da migração 002: Remover chaves estrangeiras com DELETE CASCADE

DO $$
BEGIN
    -- Remover foreign key constraints das tabelas whatsmeow
    
    -- whatsmeow_device
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_device_session') THEN
        ALTER TABLE whatsmeow_device DROP CONSTRAINT fk_whatsmeow_device_session;
    END IF;

    -- whatsmeow_identity_keys
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_identity_keys_session') THEN
        ALTER TABLE whatsmeow_identity_keys DROP CONSTRAINT fk_whatsmeow_identity_keys_session;
    END IF;

    -- whatsmeow_pre_keys
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_pre_keys_session') THEN
        ALTER TABLE whatsmeow_pre_keys DROP CONSTRAINT fk_whatsmeow_pre_keys_session;
    END IF;

    -- whatsmeow_sessions
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_sessions_session') THEN
        ALTER TABLE whatsmeow_sessions DROP CONSTRAINT fk_whatsmeow_sessions_session;
    END IF;

    -- whatsmeow_sender_keys
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_sender_keys_session') THEN
        ALTER TABLE whatsmeow_sender_keys DROP CONSTRAINT fk_whatsmeow_sender_keys_session;
    END IF;

    -- whatsmeow_app_state_sync_keys
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_app_state_sync_keys_session') THEN
        ALTER TABLE whatsmeow_app_state_sync_keys DROP CONSTRAINT fk_whatsmeow_app_state_sync_keys_session;
    END IF;

    -- whatsmeow_app_state_version
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_app_state_version_session') THEN
        ALTER TABLE whatsmeow_app_state_version DROP CONSTRAINT fk_whatsmeow_app_state_version_session;
    END IF;

    -- whatsmeow_app_state_mutation_macs
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_app_state_mutation_macs_session') THEN
        ALTER TABLE whatsmeow_app_state_mutation_macs DROP CONSTRAINT fk_whatsmeow_app_state_mutation_macs_session;
    END IF;

    -- whatsmeow_contacts
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_contacts_session') THEN
        ALTER TABLE whatsmeow_contacts DROP CONSTRAINT fk_whatsmeow_contacts_session;
    END IF;

    -- whatsmeow_chat_settings
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_chat_settings_session') THEN
        ALTER TABLE whatsmeow_chat_settings DROP CONSTRAINT fk_whatsmeow_chat_settings_session;
    END IF;

    -- Remover coluna our_jid da whatsmeow_device se foi adicionada
    IF EXISTS (SELECT 1 FROM information_schema.columns 
              WHERE table_name = 'whatsmeow_device' AND column_name = 'our_jid') THEN
        ALTER TABLE whatsmeow_device DROP COLUMN our_jid;
    END IF;

    RAISE NOTICE 'Foreign key constraints removed successfully';
END;
$$ LANGUAGE plpgsql;
