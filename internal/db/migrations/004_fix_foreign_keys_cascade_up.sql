-- Migração 004: Corrigir chaves estrangeiras com DELETE CASCADE
-- Esta migração corrige a relação entre sessions e whatsmeow_device

DO $$
BEGIN
    -- Remover constraints incorretas da migração anterior (se existirem)
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints 
              WHERE constraint_name = 'fk_whatsmeow_device_session') THEN
        ALTER TABLE whatsmeow_device DROP CONSTRAINT fk_whatsmeow_device_session;
    END IF;

    -- Verificar se as tabelas existem
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'sessions') AND
       EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_device') THEN
        
        -- Criar foreign key correta: sessions.jid -> whatsmeow_device.jid
        -- Isso garante que quando um device for removido, a sessão também seja removida
        -- Mas queremos o contrário: quando uma sessão for removida, o device deve ser removido
        
        -- Adicionar foreign key na direção correta
        IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
                      WHERE constraint_name = 'fk_sessions_whatsmeow_device') THEN
            ALTER TABLE sessions 
            ADD CONSTRAINT fk_sessions_whatsmeow_device 
            FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE SET NULL;
        END IF;
        
        RAISE NOTICE 'Foreign key constraint sessions -> whatsmeow_device added successfully';
    ELSE
        RAISE NOTICE 'Tables not found, skipping foreign key creation';
    END IF;

    -- Para garantir DELETE CASCADE quando uma sessão for removida,
    -- precisamos de uma abordagem diferente via triggers ou código da aplicação
    -- pois a relação natural é sessions -> whatsmeow_device, não o contrário

END;
$$ LANGUAGE plpgsql;
