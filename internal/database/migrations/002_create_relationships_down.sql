-- +goose Down
-- Reverter relacionamentos com tabelas WhatsApp

DO $$
BEGIN
    -- Remover foreign keys das tabelas whatsmeow
    
    -- 1. whatsmeow_device
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_device_session') THEN
        ALTER TABLE whatsmeow_device DROP CONSTRAINT fk_whatsmeow_device_session;
    END IF;

    -- 2. whatsmeow_identity_keys
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_identity_keys_session') THEN
        ALTER TABLE whatsmeow_identity_keys DROP CONSTRAINT fk_whatsmeow_identity_keys_session;
    END IF;

    -- 3. whatsmeow_pre_keys
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_pre_keys_session') THEN
        ALTER TABLE whatsmeow_pre_keys DROP CONSTRAINT fk_whatsmeow_pre_keys_session;
    END IF;

    -- 4. whatsmeow_sessions
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_sessions_session') THEN
        ALTER TABLE whatsmeow_sessions DROP CONSTRAINT fk_whatsmeow_sessions_session;
    END IF;

    -- 5. whatsmeow_sender_keys
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_sender_keys_session') THEN
        ALTER TABLE whatsmeow_sender_keys DROP CONSTRAINT fk_whatsmeow_sender_keys_session;
    END IF;

    -- 6. whatsmeow_app_state_sync_keys
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_app_state_sync_keys_session') THEN
        ALTER TABLE whatsmeow_app_state_sync_keys DROP CONSTRAINT fk_whatsmeow_app_state_sync_keys_session;
    END IF;

    -- 7. whatsmeow_app_state_version
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_app_state_version_session') THEN
        ALTER TABLE whatsmeow_app_state_version DROP CONSTRAINT fk_whatsmeow_app_state_version_session;
    END IF;

    -- 8. whatsmeow_app_state_mutation_macs
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_app_state_mutation_macs_session') THEN
        ALTER TABLE whatsmeow_app_state_mutation_macs DROP CONSTRAINT fk_whatsmeow_app_state_mutation_macs_session;
    END IF;

    -- 9. whatsmeow_contacts
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_contacts_session') THEN
        ALTER TABLE whatsmeow_contacts DROP CONSTRAINT fk_whatsmeow_contacts_session;
    END IF;

    -- 10. whatsmeow_chat_settings
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_chat_settings_session') THEN
        ALTER TABLE whatsmeow_chat_settings DROP CONSTRAINT fk_whatsmeow_chat_settings_session;
    END IF;

    RAISE NOTICE 'WhatsApp relationships removed successfully';
END;
$$ LANGUAGE plpgsql;