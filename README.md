# ZeMeow - Sistema de API Multisessão WhatsApp

Sistema backend completo em Go para gerenciamento de múltiplas sessões WhatsApp, utilizando a biblioteca `whatsmeow` com integração PostgreSQL e arquitetura RESTful.

## 🚀 Funcionalidades Principais

- **Multisessão WhatsApp**: Gerenciamento de múltiplas sessões independentes
- **API REST Simplificada**: Endpoints limpos sem prefixos desnecessários
- **Autenticação por API Key**: Sistema simplificado com chave global admin e chaves por sessão
- **Integração com WhatsApp**: Usando `go.mau.fi/whatsmeow` com sqlstore
- **Persistência PostgreSQL**: Armazenamento confiável de dados de sessão
- **Geração Automática de API Keys**: Sistema automático ou manual de chaves
- **Logs Estruturados**: Sistema de logging completo com zerolog

## 📋 Pré-requisitos

- Go 1.23 ou superior
- PostgreSQL 15+

## 🛠️ Instalação e Configuração

### 1. Clonar o Repositório
```bash
git clone <repository-url>
cd zemeow
```

### 2. Configurar Variáveis de Ambiente
```bash
cp .env.example .env
# Editar .env com suas configurações
```

### 3. Iniciar Banco de Dados (Docker)
```bash
docker-compose up -d postgres
```

### 4. Instalar Dependências
```bash
go mod download
```

### 5. Executar a Aplicação
As migrações do banco de dados são executadas automaticamente ao iniciar a aplicação:

```bash
go run cmd/zemeow/main.go
```

## 📡 Endpoints da API

### Criação de Sessão (Admin)
```bash
POST /sessions
Authorization: Bearer YOUR_ADMIN_API_KEY

{
  "name": "Minha Sessão",
  "api_key": "opcional-custom-key",
  "webhook": {
    "url": "https://exemplo.com/webhook"
  }
}
```

### Listar Sessões (Admin)
```bash
GET /sessions
Authorization: Bearer YOUR_ADMIN_API_KEY
```

### Obter Informações da Sessão
```bash
GET /sessions/{sessionId}
Authorization: Bearer SESSION_API_KEY
```

### Conectar Sessão ao WhatsApp
```bash
POST /sessions/{sessionId}/connect
Authorization: Bearer SESSION_API_KEY
```

### Obter QR Code
```bash
GET /sessions/{sessionId}/qr
Authorization: Bearer SESSION_API_KEY
```

### Verificar Status da Sessão
```bash
GET /sessions/{sessionId}/status
Authorization: Bearer SESSION_API_KEY
```

### Deletar Sessão
```bash
DELETE /sessions/{sessionId}
Authorization: Bearer SESSION_API_KEY
```

## 🔐 Sistema de Autenticação

### API Key Global (Admin)
- Definida em `ADMIN_API_KEY` no arquivo `.env`
- Permite criar e gerenciar todas as sessões
- Acesso completo ao sistema

### API Key por Sessão
- Gerada automaticamente na criação da sessão
- Pode ser personalizada pelo usuário
- Acesso restrito apenas à sessão específica

### Uso nos Headers
```bash
Authorization: Bearer YOUR_API_KEY
# ou
X-API-Key: YOUR_API_KEY
```

## ⚙️ Configurações (.env)

```bash
# Banco de Dados
DATABASE_URL=postgres://zemeow:zemeow123@localhost:5432/zemeow
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=zemeow
POSTGRES_USER=zemeow
POSTGRES_PASSWORD=zemeow123

# Servidor
SERVER_PORT=8080

# Chave Admin (ALTERE PARA UM VALOR SEGURO)
ADMIN_API_KEY=your_secure_admin_api_key_here

# Logging
LOG_LEVEL=info
LOG_PRETTY=true
```

## 🗃️ Estrutura do Banco

O sistema cria automaticamente as seguintes tabelas:
- `sessions`: Dados das sessões
- `whatsmeow_*`: Tabelas do whatsmeow para armazenamento do WhatsApp

## 🔄 Fluxo de Uso

1. **Configurar** variáveis de ambiente
2. **Iniciar** o servidor: `go run cmd/zemeow/main.go`
3. **Criar sessão** usando Admin API Key
4. **Conectar** sessão ao WhatsApp
5. **Obter QR Code** para autenticação
6. **Usar** a sessão com sua API Key específica

## 📂 Estrutura do Projeto

```
zemeow/
├── cmd/zemeow/main.go              # Ponto de entrada
├── internal/
│   ├── api/                        # Camada HTTP
│   │   ├── handlers/              # Handlers REST
│   │   ├── middleware/            # Middlewares
│   │   └── server.go              # Servidor HTTP
│   ├── config/                    # Configurações
│   ├── db/                        # Banco de dados
│   │   ├── migrations/            # Migrations
│   │   ├── models/               # Modelos
│   │   └── repositories/         # Repositórios
│   ├── logger/                    # Sistema de logs
│   └── service/                   # Lógica de negócio
│       ├── session/              # Gerenciamento de sessões
│       └── meow/                 # Integração WhatsApp
├── .env.example                   # Exemplo de configuração
├── docker-compose.yml            # Docker Compose
└── go.mod                        # Dependências Go
```

## 🔧 Desenvolvimento

### Executar em Modo Desenvolvimento
```bash
go run cmd/zemeow/main.go
```

### Build para Produção
```bash
go build -o zemeow cmd/zemeow/main.go
```

### Docker
```bash
docker-compose up -d
```

## 📝 Logs

O sistema utiliza logs estruturados com diferentes níveis de severidade. Em ambiente de desenvolvimento, os logs são formatados para melhor leitura.

## 🔄 Migrações de Banco de Dados

As migrações são executadas automaticamente ao iniciar a aplicação. Não é necessário executar comandos manuais para migrações.

## 🛡️ Segurança

- Todas as API Keys são tratadas como segredos
- Comunicação com o banco de dados pode ser configurada com SSL
- Logs não contêm informações sensíveis

## 📈 Monitoramento

O sistema inclui métricas básicas de saúde e estatísticas de uso das sessões.