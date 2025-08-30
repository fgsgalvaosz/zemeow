-- Migração 002: Adicionar chaves estrangeiras com DELETE CASCADE para tabelas whatsmeow
-- Esta migração adiciona relacionamentos entre a tabela sessions e as tabelas do whatsmeow
-- com DELETE CASCADE para garantir limpeza automática quando uma sessão for removida

-- Aguardar que as tabelas do whatsmeow sejam criadas pelo sqlstore
-- Verificar se as tabelas existem antes de adicionar as constraints

DO $$
BEGIN
    -- Adicionar coluna our_jid na tabela whatsmeow_device se não existir
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_device') THEN
        -- Verificar se a coluna our_jid já existe
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                      WHERE table_name = 'whatsmeow_device' AND column_name = 'our_jid') THEN
            ALTER TABLE whatsmeow_device ADD COLUMN our_jid TEXT;
        END IF;
        
        -- Adicionar foreign key constraint com CASCADE
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_device_session') THEN
            ALTER TABLE whatsmeow_device 
            ADD CONSTRAINT fk_whatsmeow_device_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE;
        END IF;
    END IF;

    -- Adicionar foreign keys para whatsmeow_identity_keys
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_identity_keys') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_identity_keys_session') THEN
            ALTER TABLE whatsmeow_identity_keys 
            ADD CONSTRAINT fk_whatsmeow_identity_keys_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE;
        END IF;
    END IF;

    -- Adicionar foreign keys para whatsmeow_pre_keys
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_pre_keys') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_pre_keys_session') THEN
            ALTER TABLE whatsmeow_pre_keys 
            ADD CONSTRAINT fk_whatsmeow_pre_keys_session 
            FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE;
        END IF;
    END IF;

    -- Adicionar foreign keys para whatsmeow_sessions
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_sessions') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_sessions_session') THEN
            ALTER TABLE whatsmeow_sessions 
            ADD CONSTRAINT fk_whatsmeow_sessions_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE;
        END IF;
    END IF;

    -- Adicionar foreign keys para whatsmeow_sender_keys
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_sender_keys') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_sender_keys_session') THEN
            ALTER TABLE whatsmeow_sender_keys 
            ADD CONSTRAINT fk_whatsmeow_sender_keys_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE;
        END IF;
    END IF;

    -- Adicionar foreign keys para whatsmeow_app_state_sync_keys
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_sync_keys') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_app_state_sync_keys_session') THEN
            ALTER TABLE whatsmeow_app_state_sync_keys 
            ADD CONSTRAINT fk_whatsmeow_app_state_sync_keys_session 
            FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE;
        END IF;
    END IF;

    -- Adicionar foreign keys para whatsmeow_app_state_version
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_version') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_app_state_version_session') THEN
            ALTER TABLE whatsmeow_app_state_version 
            ADD CONSTRAINT fk_whatsmeow_app_state_version_session 
            FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE;
        END IF;
    END IF;

    -- Adicionar foreign keys para whatsmeow_app_state_mutation_macs
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_mutation_macs') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_app_state_mutation_macs_session') THEN
            ALTER TABLE whatsmeow_app_state_mutation_macs 
            ADD CONSTRAINT fk_whatsmeow_app_state_mutation_macs_session 
            FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE;
        END IF;
    END IF;

    -- Adicionar foreign keys para whatsmeow_contacts
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_contacts') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_contacts_session') THEN
            ALTER TABLE whatsmeow_contacts 
            ADD CONSTRAINT fk_whatsmeow_contacts_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE;
        END IF;
    END IF;

    -- Adicionar foreign keys para whatsmeow_chat_settings
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_chat_settings') THEN
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_whatsmeow_chat_settings_session') THEN
            ALTER TABLE whatsmeow_chat_settings 
            ADD CONSTRAINT fk_whatsmeow_chat_settings_session 
            FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE;
        END IF;
    END IF;

    RAISE NOTICE 'Foreign key constraints with CASCADE added successfully';
END;
$$ LANGUAGE plpgsql;
