# Relacionamentos entre Tabelas Sessions e WhatsApp

## Visão Geral

O ZeMeow implementa relacionamentos CASCADE entre as tabelas da aplicação (`sessions`) e as tabelas do WhatsApp (`whatsmeow_*`) para garantir integridade referencial e limpeza automática de dados.

## Arquitetura de Relacionamentos

### Tabela Principal: `sessions`
- Contém informações das sessões WhatsApp da aplicação
- Campo `jid` referencia `whatsmeow_device.jid`
- Quando uma sessão é deletada, todos os dados WhatsApp relacionados são removidos automaticamente

### Tabelas WhatsApp (Auto-gerenciadas)
- `whatsmeow_device` - Dispositivo principal (referenciado por sessions.jid)
- `whatsmeow_identity_keys` - Chaves de identidade (relacionadas por our_jid)
- `whatsmeow_pre_keys` - Chaves pré-compartilhadas (relacionadas por jid)
- `whatsmeow_sender_keys` - Chaves de remetente (relacionadas por our_jid)
- `whatsmeow_app_state_*` - Estado da aplicação (relacionadas por jid)
- `whatsmeow_contacts` - Contatos (relacionadas por our_jid)
- `whatsmeow_chat_settings` - Configurações de chat (relacionadas por our_jid)

## Foreign Keys e Cascades

### Relacionamento Principal
```sql
ALTER TABLE sessions 
ADD CONSTRAINT fk_sessions_whatsmeow_device 
FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) 
ON DELETE CASCADE ON UPDATE CASCADE;
```

**Comportamento:**
- Quando `whatsmeow_device` é deletado → `sessions` correspondente é deletada
- Quando `whatsmeow_device.jid` é atualizado → `sessions.jid` é atualizado automaticamente

### Cascata de Limpeza
Quando uma sessão é removida:
1. `sessions` é deletada (manualmente ou via cascade)
2. `whatsmeow_device` correspondente é removido
3. Todas as tabelas relacionadas são limpas automaticamente pelo whatsmeow

## Índices Otimizados

### Índices para Performance
```sql
-- Dispositivos
CREATE INDEX idx_whatsmeow_device_jid_lookup ON whatsmeow_device(jid);
CREATE INDEX idx_whatsmeow_device_registration ON whatsmeow_device(registration_id);

-- Chaves de Identidade
CREATE INDEX idx_whatsmeow_identity_our_jid ON whatsmeow_identity_keys(our_jid);
CREATE INDEX idx_whatsmeow_identity_their_id ON whatsmeow_identity_keys(their_id);

-- Chaves Pré-compartilhadas
CREATE INDEX idx_whatsmeow_prekeys_jid ON whatsmeow_pre_keys(jid);
CREATE INDEX idx_whatsmeow_prekeys_uploaded ON whatsmeow_pre_keys(jid, uploaded);

-- Contatos
CREATE INDEX idx_whatsmeow_contacts_our_jid ON whatsmeow_contacts(our_jid);
CREATE INDEX idx_whatsmeow_contacts_names ON whatsmeow_contacts(our_jid, first_name, full_name);

-- Configurações de Chat
CREATE INDEX idx_whatsmeow_chat_settings ON whatsmeow_chat_settings(our_jid, chat_jid);
CREATE INDEX idx_whatsmeow_chat_muted ON whatsmeow_chat_settings(our_jid, muted_until) WHERE muted_until > 0;
```

### Índices Condicionais
Para otimizar queries específicas:
```sql
-- Apenas chats mutados
WHERE muted_until > 0

-- Apenas chats fixados
WHERE pinned = true

-- Apenas chats arquivados
WHERE archived = true

-- Apenas contatos com push_name
WHERE push_name != ''
```

## Fluxo de Inicialização

### 1. Inicialização do Banco
```go
// 1. Conectar ao PostgreSQL
dbConn, err := db.Connect(cfg)

// 2. Executar migrações da aplicação (tabela sessions)
err := dbConn.Migrate()
```

### 2. Inicialização WhatsApp
```go
// 3. Criar WhatsApp SQL Store (cria tabelas whatsmeow_*)
sqlStore := dbConn.GetSQLStore()
// Internamente executa: container.Upgrade(ctx)
```

### 3. Criação de Relacionamentos
```go
// 4. Re-executar migrações para criar relacionamentos
err := dbConn.Migrate()
// Migrações 3, 4, 5 criam foreign keys e índices
```

## Migrações Específicas

### Migração 3: Relacionamentos
- Cria foreign key entre `sessions.jid` e `whatsmeow_device.jid`
- Usa função PL/pgSQL para verificar se tabelas existem
- Executa apenas se tabelas whatsmeow estiverem criadas

### Migração 4: Índices Básicos
- Cria índices para tabelas principais (device, identity_keys, pre_keys, sender_keys)
- Usa `CREATE INDEX CONCURRENTLY` para não bloquear

### Migração 5: Índices Avançados
- Cria índices para app_state e contacts
- Inclui índices condicionais para otimização

## Operações de Limpeza

### Remoção de Sessão
```go
// Remover sessão (limpa tudo automaticamente)
err := sessionRepo.DeleteByName(sessionName)
```

**Resultado:**
1. `sessions` é deletada
2. Foreign key cascade remove `whatsmeow_device`
3. WhatsApp library limpa tabelas relacionadas automaticamente

### Limpeza Manual
```sql
-- Limpar dados órfãos (se necessário)
DELETE FROM whatsmeow_identity_keys 
WHERE our_jid NOT IN (SELECT jid FROM whatsmeow_device WHERE jid IS NOT NULL);

DELETE FROM whatsmeow_pre_keys 
WHERE jid NOT IN (SELECT jid FROM whatsmeow_device WHERE jid IS NOT NULL);
```

## Monitoramento

### Verificar Integridade
```sql
-- Verificar sessions sem device
SELECT s.name, s.jid
FROM sessions s
LEFT JOIN whatsmeow_device d ON s.jid = d.jid
WHERE s.jid IS NOT NULL AND d.jid IS NULL;

-- Verificar devices sem session
SELECT d.jid 
FROM whatsmeow_device d 
LEFT JOIN sessions s ON d.jid = s.jid 
WHERE s.jid IS NULL;
```

### Estatísticas de Uso
```sql
-- Contagem por tabela
SELECT 
    'sessions' as table_name, COUNT(*) as count FROM sessions
UNION ALL
SELECT 
    'whatsmeow_device', COUNT(*) FROM whatsmeow_device
UNION ALL
SELECT 
    'whatsmeow_contacts', COUNT(*) FROM whatsmeow_contacts;
```

## Troubleshooting

### Problemas Comuns

1. **Foreign key violation**
   - Causa: Tentativa de inserir session com JID inexistente
   - Solução: Garantir que device existe antes de criar session

2. **Índices não criados**
   - Causa: Tabelas whatsmeow não existiam durante migração
   - Solução: Re-executar migrações após upgrade do whatsmeow

3. **Dados órfãos**
   - Causa: Remoção manual sem respeitar foreign keys
   - Solução: Usar queries de limpeza manual

### Debug
```sql
-- Ver todas as constraints
SELECT conname, contype, confupdtype, confdeltype 
FROM pg_constraint 
WHERE conrelid = 'sessions'::regclass;

-- Ver todos os índices
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename LIKE 'whatsmeow_%' 
ORDER BY tablename, indexname;
```
