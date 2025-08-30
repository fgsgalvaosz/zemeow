# zemeow
>>>>>>> d5625307bc3f26b192274cffd20d08b75e1c9939
# ZeMeow - API Multisessão WhatsApp

Sistema completo de multisessão WhatsApp desenvolvido em Go, permitindo gerenciar múltiplas sessões simultaneamente com integração ao PostgreSQL.

## 🚀 Características

- **Multisessão**: Gerenciamento de múltiplas sessões WhatsApp simultâneas
- **API REST**: Interface completa para gerenciamento via HTTP
- **PostgreSQL**: Persistência robusta com sqlstore do whatsmeow
- **Autenticação**: Sistema seguro baseado em tokens de sessão
- **QR Code**: Autenticação via QR Code e Pair by Phone
- **Webhooks**: Sistema de eventos em tempo real
- **Docker**: Containerização completa com docker-compose
- **Logs Estruturados**: Sistema de logging centralizado com zerolog

## 🛠️ Tecnologias

- **Go 1.23+** - Linguagem principal
- **Fiber v2** - Framework web
- **WhatsApp (whatsmeow)** - Cliente WhatsApp oficial
- **PostgreSQL 15** - Banco de dados
- **Docker & Docker Compose** - Containerização
- **Zerolog** - Logging estruturado

## 📋 Pré-requisitos

- Go 1.23 ou superior
- Docker e Docker Compose
- PostgreSQL 15 (ou via Docker)

## 🚀 Instalação

1. **Clone o repositório:**
```bash
git clone https://github.com/fgsgalvaosz/zemeow.git
cd zemeow
```

2. **Configure as variáveis de ambiente:**
```bash
cp .env.example .env
# Edite o arquivo .env com suas configurações
```

3. **Inicie os serviços com Docker:**
```bash
docker-compose up -d
```

4. **Instale as dependências Go:**
```bash
go mod download
```

5. **Execute as migrações:**
```bash
go run cmd/zemeow/main.go migrate
```

6. **Inicie a aplicação:**
```bash
go run cmd/zemeow/main.go
```

## 📖 API Endpoints

### Sessões
- `POST /api/v1/sessions` - Criar nova sessão
- `GET /api/v1/sessions` - Listar sessões
- `GET /api/v1/sessions/{id}` - Obter detalhes da sessão
- `PUT /api/v1/sessions/{id}` - Atualizar sessão
- `DELETE /api/v1/sessions/{id}` - Remover sessão

### Conexão WhatsApp
- `POST /api/v1/sessions/{id}/connect` - Conectar sessão
- `POST /api/v1/sessions/{id}/disconnect` - Desconectar sessão
- `GET /api/v1/sessions/{id}/qr` - Obter QR Code
- `POST /api/v1/sessions/{id}/pairphone` - Pair por telefone
- `GET /api/v1/sessions/{id}/status` - Status da conexão

### Autenticação
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/refresh` - Renovar token
- `GET /api/v1/auth/validate` - Validar token

## 🏗️ Arquitetura

```
internal/
├── api/           # Handlers e rotas HTTP
├── config/        # Configurações da aplicação
├── db/            # Modelos e repositórios
├── logger/        # Sistema de logging
└── service/       # Lógica de negócio
    ├── auth/      # Autenticação
    ├── meow/      # Cliente WhatsApp
    ├── session/   # Gerenciamento de sessões
    └── webhook/   # Sistema de webhooks
```

## 🔧 Configuração

O projeto utiliza variáveis de ambiente para configuração. Principais variáveis:

```env
# Banco de dados
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=zemeow
POSTGRES_USER=zemeow
POSTGRES_PASSWORD=zemeow123

# Servidor
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Autenticação
ADMIN_TOKEN=your_admin_token_here
JWT_SECRET=your_jwt_secret_here

# Logging
LOG_LEVEL=info
LOG_PRETTY=true
```

## 🤝 Contribuindo

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/nova-feature`)
3. Commit suas mudanças (`git commit -am 'Add nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Abra um Pull Request

## 📄 Licença

Este projeto está sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.

## ⚠️ Aviso Legal

Este projeto utiliza a biblioteca não oficial `whatsmeow` para integração com WhatsApp. O uso deve estar em conformidade com os Termos de Serviço do WhatsApp.

## 📞 Suporte

Para dúvidas e suporte, abra uma [issue](https://github.com/fgsgalvaosz/zemeow/issues) no GitHub.
=======
# zemeow
>>>>>>> d5625307bc3f26b192274cffd20d08b75e1c9939
