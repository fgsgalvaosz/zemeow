-- +goose Up
-- Criar relacionamentos com tabelas WhatsApp
-- Esta migração será executada após as tabelas do whatsmeow serem criadas

DO $$
BEGIN
    -- Aguardar que as tabelas do whatsmeow sejam criadas
    -- Esta migração adiciona foreign keys com CASCADE para limpeza automática
    
    -- 1. whatsmeow_device
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_device') THEN
        -- Adicionar coluna our_jid se não existir
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                      WHERE table_name = 'whatsmeow_device' AND column_name = 'our_jid') THEN
            ALTER TABLE whatsmeow_device ADD COLUMN our_jid TEXT;
        END IF;
        
        -- Adicionar foreign key
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_device_session') THEN
            ALTER TABLE whatsmeow_device 
            ADD CONSTRAINT fk_whatsmeow_device_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    END IF;

    -- 2. whatsmeow_identity_keys
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_identity_keys') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_identity_keys_session') THEN
            ALTER TABLE whatsmeow_identity_keys 
            ADD CONSTRAINT fk_whatsmeow_identity_keys_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    END IF;

    -- 3. whatsmeow_pre_keys
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_pre_keys') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_pre_keys_session') THEN
            ALTER TABLE whatsmeow_pre_keys 
            ADD CONSTRAINT fk_whatsmeow_pre_keys_session 
            FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    END IF;

    -- 4. whatsmeow_sessions
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_sessions') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_sessions_session') THEN
            ALTER TABLE whatsmeow_sessions 
            ADD CONSTRAINT fk_whatsmeow_sessions_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    END IF;

    -- 5. whatsmeow_sender_keys
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_sender_keys') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_sender_keys_session') THEN
            ALTER TABLE whatsmeow_sender_keys 
            ADD CONSTRAINT fk_whatsmeow_sender_keys_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    END IF;

    -- 6. whatsmeow_app_state_sync_keys
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_sync_keys') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_app_state_sync_keys_session') THEN
            ALTER TABLE whatsmeow_app_state_sync_keys 
            ADD CONSTRAINT fk_whatsmeow_app_state_sync_keys_session 
            FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    END IF;

    -- 7. whatsmeow_app_state_version
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_version') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_app_state_version_session') THEN
            ALTER TABLE whatsmeow_app_state_version 
            ADD CONSTRAINT fk_whatsmeow_app_state_version_session 
            FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    END IF;

    -- 8. whatsmeow_app_state_mutation_macs
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_mutation_macs') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_app_state_mutation_macs_session') THEN
            ALTER TABLE whatsmeow_app_state_mutation_macs 
            ADD CONSTRAINT fk_whatsmeow_app_state_mutation_macs_session 
            FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    END IF;

    -- 9. whatsmeow_contacts
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_contacts') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_contacts_session') THEN
            ALTER TABLE whatsmeow_contacts 
            ADD CONSTRAINT fk_whatsmeow_contacts_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    END IF;

    -- 10. whatsmeow_chat_settings
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_chat_settings') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_chat_settings_session') THEN
            ALTER TABLE whatsmeow_chat_settings 
            ADD CONSTRAINT fk_whatsmeow_chat_settings_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
        END IF;
    END IF;

    RAISE NOTICE 'WhatsApp relationships created successfully with deferrable constraints';
END;
$$ LANGUAGE plpgsql;