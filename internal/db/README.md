# Sistema de Banco de Dados ZeMeow

## Visão Geral

O ZeMeow utiliza um sistema híbrido de gerenciamento de banco de dados que combina:

1. **Migrações da Aplicação**: Para tabelas específicas do ZeMeow (como `sessions`)
2. **Auto-upgrade do WhatsApp**: Para tabelas do WhatsApp gerenciadas automaticamente pela biblioteca `whatsmeow`

## Arquitetura

### Tabelas da Aplicação
- `sessions` - Dados das sessões WhatsApp
- `schema_migrations` - Controle de versões das migrações

### Tabelas do WhatsApp (Auto-gerenciadas)
- `whatsmeow_device` - Dispositivos WhatsApp
- `whatsmeow_identity_keys` - Chaves de identidade
- `whatsmeow_pre_keys` - Chaves pré-compartilhadas
- `whatsmeow_sender_keys` - Chaves de remetente
- `whatsmeow_app_state_*` - Estado da aplicação WhatsApp
- `whatsmeow_contacts` - Contatos
- `whatsmeow_chat_settings` - Configurações de chat
- E outras tabelas conforme necessário

## Como Funciona

### 1. Inicialização Automática

Quando a aplicação inicia:

```go
// 1. Conecta ao banco
dbConn, err := db.Connect(cfg)

// 2. Executa migrações da aplicação
err := dbConn.Migrate()

// 3. Cria índices otimizados
err := dbConn.CreateIndexes()

// 4. Aplica otimizações PostgreSQL
err := dbConn.OptimizeForWhatsApp()

// 5. Verifica se tudo está funcionando
err := dbConn.VerifySetup()
```

### 2. WhatsApp Store Auto-upgrade

As tabelas do WhatsApp são gerenciadas automaticamente:

```go
// O container do whatsmeow faz upgrade automático
container := sqlstore.NewWithDB(db.DB, "postgres", logger)
err := container.Upgrade(ctx) // Cria/atualiza tabelas automaticamente
```

## Migrações da Aplicação

### Estrutura

```go
type Migration struct {
    Version     int
    Description string
    Up          string    // SQL para aplicar
    Down        string   // SQL para reverter
}
```

### Adicionando Nova Migração

1. Edite `internal/db/migrations/migrations.go`
2. Adicione nova migração ao slice:

```go
{
    Version:     3,
    Description: "Add new feature table",
    Up: `
        CREATE TABLE new_feature (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            name VARCHAR(255) NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );
    `,
    Down: `
        DROP TABLE IF EXISTS new_feature;
    `,
},
```

### Controle de Versão

- Migrações são aplicadas automaticamente na inicialização
- Tabela `schema_migrations` controla quais foram aplicadas
- Migrações são executadas em ordem crescente de versão
- Migrações já aplicadas são ignoradas

## Otimizações

### Índices Automáticos

O sistema cria automaticamente índices otimizados:

```sql
-- Para tabela sessions
CREATE INDEX idx_sessions_status_created ON sessions(status, created_at);
CREATE INDEX idx_sessions_jid_status ON sessions(jid, status);
CREATE INDEX idx_sessions_last_activity ON sessions(last_activity);
```

### Configurações PostgreSQL

Aplica configurações otimizadas para WhatsApp:

```sql
SET statement_timeout = '30s';
SET lock_timeout = '10s';
SET autovacuum_vacuum_scale_factor = 0.1;
```

## Desenvolvimento

### Testando Migrações

```bash
# Executar aplicação (migrações automáticas)
go run cmd/zemeow/main.go

# Verificar logs para confirmar migrações
# Logs mostrarão: "Application migrations completed successfully"
```

### Rollback Manual

Se necessário, use ferramentas como `migrate` ou SQL direto:

```sql
-- Ver migrações aplicadas
SELECT * FROM schema_migrations ORDER BY version;

-- Rollback manual (cuidado!)
DELETE FROM schema_migrations WHERE version = 2;
-- Execute o SQL de Down da migração manualmente
```

## Monitoramento

### Health Check

```go
// Verificar saúde do banco
err := db.Health()
```

### Estatísticas

```go
// Obter estatísticas de conexão
stats := db.GetStats()
```

### Logs

O sistema produz logs estruturados:

```
INFO Database connection established successfully
INFO Application migrations completed successfully  
INFO Application indexes created successfully
INFO PostgreSQL optimizations applied successfully
INFO Database setup verification completed successfully
```

## Troubleshooting

### Problemas Comuns

1. **Migrações falhando**
   - Verificar logs de erro
   - Confirmar permissões do usuário PostgreSQL
   - Verificar sintaxe SQL

2. **Tabelas WhatsApp não criadas**
   - Verificar se `container.Upgrade()` foi chamado
   - Verificar logs do whatsmeow
   - Confirmar versão da biblioteca

3. **Performance lenta**
   - Verificar se índices foram criados
   - Verificar configurações PostgreSQL
   - Analisar queries com `EXPLAIN`

### Debug

Para debug detalhado, ajuste o nível de log:

```bash
LOG_LEVEL=debug go run cmd/zemeow/main.go
```

## Segurança

- Migrações são executadas com as permissões do usuário da aplicação
- Não há SQL dinâmico - apenas queries pré-definidas
- Transações garantem consistência
- Rollback automático em caso de erro
