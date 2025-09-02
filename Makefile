# ZeMeow WhatsApp API - Makefile
# Facilita o desenvolvimento e deploy da aplicação

.PHONY: help dev prod build clean logs test docs swagger migrate

# Variáveis
DOCKER_COMPOSE_DEV = docker compose -f docker-compose.dev.yml
DOCKER_COMPOSE_PROD = docker compose -f docker-compose.yml
GO_FILES = $(shell find . -name "*.go" -type f)

# Cores para output
GREEN = \033[0;32m
YELLOW = \033[1;33m
RED = \033[0;31m
NC = \033[0m # No Color

## 📋 Ajuda
help: ## Mostra esta ajuda
	@echo "$(GREEN)ZeMeow WhatsApp API - Comandos Disponíveis$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "$(YELLOW)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## 🚀 Desenvolvimento
dev: ## Inicia ambiente de desenvolvimento (apenas dependências)
	@echo "$(GREEN)🚀 Iniciando ambiente de desenvolvimento...$(NC)"
	$(DOCKER_COMPOSE_DEV) up -d
	@echo "$(GREEN)✅ Dependências iniciadas:$(NC)"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - MinIO: localhost:9000 (console: localhost:9001)"
	@echo "  - Redis: localhost:6379"
	@echo "  - Webhook Tester: localhost:8081"
	@echo ""
	@echo "$(YELLOW)💡 Para iniciar a API:$(NC) make run"

dev-stop: ## Para ambiente de desenvolvimento
	@echo "$(YELLOW)🛑 Parando ambiente de desenvolvimento...$(NC)"
	$(DOCKER_COMPOSE_DEV) down

dev-logs: ## Mostra logs do ambiente de desenvolvimento
	$(DOCKER_COMPOSE_DEV) logs -f

## 🏭 Produção
prod: ## Inicia ambiente completo (com ZeMeow)
	@echo "$(GREEN)🏭 Iniciando ambiente de produção...$(NC)"
	$(DOCKER_COMPOSE_PROD) up -d
	@echo "$(GREEN)✅ Ambiente completo iniciado:$(NC)"
	@echo "  - ZeMeow API: localhost:8080"
	@echo "  - PostgreSQL: localhost:5432"
	@echo "  - MinIO: localhost:9000 (console: localhost:9001)"
	@echo "  - Redis: localhost:6379"
	@echo "  - Swagger UI: localhost:8080/swagger/index.html"

prod-stop: ## Para ambiente de produção
	@echo "$(YELLOW)🛑 Parando ambiente de produção...$(NC)"
	$(DOCKER_COMPOSE_PROD) down

prod-logs: ## Mostra logs do ambiente de produção
	$(DOCKER_COMPOSE_PROD) logs -f

## 🚀 Stack Produção (Traefik)
stack-deploy: ## Deploy da stack no Docker Swarm (Portainer)
	@echo "$(GREEN)🚀 Preparando stack para deploy...$(NC)"
	@echo "$(YELLOW)📋 Arquivo para Portainer: docker-compose.prod.yml$(NC)"
	@echo "$(YELLOW)🌐 Domínios configurados:$(NC)"
	@echo "  - ZeMeow API: https://zemeow.gacont.com.br"
	@echo "  - MinIO S3: https://zs3.gacont.com.br"
	@echo "  - MinIO Console: https://zminio.gacont.com.br"
	@echo ""
	@echo "$(GREEN)📝 Passos para deploy no Portainer:$(NC)"
	@echo "1. Acesse Portainer > Stacks > Add Stack"
	@echo "2. Cole o conteúdo de docker-compose.prod.yml"
	@echo "3. Ajuste os domínios conforme necessário"
	@echo "4. Deploy da stack"

stack-volumes: ## Cria volumes necessários para a stack
	@echo "$(GREEN)📦 Criando volumes para a stack...$(NC)"
	docker volume create zemeow_postgres_data
	docker volume create zemeow_redis_data
	docker volume create zemeow_minio_data
	docker volume create zemeow_sessions_data
	docker volume create zemeow_logs_data
	@echo "$(GREEN)✅ Volumes criados com sucesso$(NC)"

## 🔧 Desenvolvimento Local
run: ## Executa a API localmente (requer 'make dev' primeiro)
	@echo "$(GREEN)🔧 Iniciando ZeMeow API localmente...$(NC)"
	go run cmd/zemeow/main.go

build: ## Compila a aplicação
	@echo "$(GREEN)🔨 Compilando ZeMeow...$(NC)"
	go build -o bin/zemeow cmd/zemeow/main.go
	@echo "$(GREEN)✅ Compilado em: bin/zemeow$(NC)"

docker-build: ## Faz build da imagem Docker
	@echo "$(GREEN)🐳 Fazendo build da imagem Docker...$(NC)"
	docker build -t felipyfgs17/zemeow:latest -t felipyfgs17/zemeow:v1.1.0 .
	@echo "$(GREEN)✅ Imagem Docker criada com sucesso$(NC)"

docker-push: ## Faz push da imagem para Docker Hub
	@echo "$(GREEN)📤 Fazendo push para Docker Hub...$(NC)"
	docker push felipyfgs17/zemeow:latest
	docker push felipyfgs17/zemeow:v1.1.0
	@echo "$(GREEN)✅ Imagem enviada para Docker Hub$(NC)"

docker-release: docker-build docker-push ## Build e push da imagem Docker

test: ## Executa os testes
	@echo "$(GREEN)🧪 Executando testes...$(NC)"
	go test -v ./...

clean: ## Limpa containers e volumes
	@echo "$(YELLOW)🧹 Limpando containers e volumes...$(NC)"
	$(DOCKER_COMPOSE_DEV) down -v
	$(DOCKER_COMPOSE_PROD) down -v
	docker system prune -f

## 📊 Banco de Dados
migrate: ## Executa migrações do banco
	@echo "$(GREEN)📊 Executando migrações...$(NC)"
	go run cmd/zemeow/main.go migrate

db-reset: ## Reseta o banco de dados
	@echo "$(RED)⚠️  Resetando banco de dados...$(NC)"
	$(DOCKER_COMPOSE_DEV) exec postgres psql -U zemeow -d zemeow -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	make migrate

## 📚 Documentação
docs: swagger ## Alias para swagger

swagger: ## Regenera documentação Swagger
	@echo "$(GREEN)📚 Regenerando documentação Swagger...$(NC)"
	/go/bin/swag init -g cmd/zemeow/main.go -o docs
	@echo "$(GREEN)✅ Documentação atualizada em: docs/$(NC)"
	@echo "$(YELLOW)💡 Acesse: http://localhost:8080/swagger/index.html$(NC)"

## 🔍 Monitoramento
status: ## Mostra status dos serviços
	@echo "$(GREEN)🔍 Status dos serviços:$(NC)"
	@echo ""
	@echo "$(YELLOW)Desenvolvimento:$(NC)"
	$(DOCKER_COMPOSE_DEV) ps
	@echo ""
	@echo "$(YELLOW)Produção:$(NC)"
	$(DOCKER_COMPOSE_PROD) ps

logs: ## Mostra logs de todos os serviços
	@echo "$(GREEN)📋 Logs dos serviços:$(NC)"
	$(DOCKER_COMPOSE_PROD) logs -f --tail=100

health: ## Verifica saúde da API
	@echo "$(GREEN)🏥 Verificando saúde da API...$(NC)"
	@curl -s http://localhost:8080/health | jq . || echo "$(RED)❌ API não está respondendo$(NC)"

## 🛠️ Utilitários
install-deps: ## Instala dependências Go
	@echo "$(GREEN)📦 Instalando dependências...$(NC)"
	go mod download
	go mod tidy

install-tools: ## Instala ferramentas de desenvolvimento
	@echo "$(GREEN)🔧 Instalando ferramentas...$(NC)"
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/air-verse/air@latest

fmt: ## Formata código Go
	@echo "$(GREEN)✨ Formatando código...$(NC)"
	go fmt ./...

lint: ## Executa linter
	@echo "$(GREEN)🔍 Executando linter...$(NC)"
	golangci-lint run

## 🎯 Comandos Rápidos
quick-start: dev run ## Inicia desenvolvimento rapidamente

full-start: prod ## Inicia ambiente completo

restart: prod-stop prod ## Reinicia ambiente de produção

restart-dev: dev-stop dev ## Reinicia ambiente de desenvolvimento

## 📱 Exemplos de Uso
examples: ## Mostra exemplos de uso da API
	@echo "$(GREEN)📱 Exemplos de uso da ZeMeow API:$(NC)"
	@echo ""
	@echo "$(YELLOW)1. Criar sessão:$(NC)"
	@echo "curl -X POST http://localhost:8080/sessions \\"
	@echo "  -H 'Authorization: Bearer test123' \\"
	@echo "  -H 'Content-Type: application/json' \\"
	@echo "  -d '{\"name\": \"minha-sessao\"}'"
	@echo ""
	@echo "$(YELLOW)2. Listar sessões:$(NC)"
	@echo "curl -H 'Authorization: Bearer test123' http://localhost:8080/sessions"
	@echo ""
	@echo "$(YELLOW)3. Conectar sessão:$(NC)"
	@echo "curl -X POST http://localhost:8080/sessions/{sessionId}/connect \\"
	@echo "  -H 'Authorization: Bearer test123'"
	@echo ""
	@echo "$(YELLOW)4. Enviar mensagem:$(NC)"
	@echo "curl -X POST http://localhost:8080/sessions/{sessionId}/send/text \\"
	@echo "  -H 'Authorization: Bearer {session_api_key}' \\"
	@echo "  -H 'Content-Type: application/json' \\"
	@echo "  -d '{\"to\": \"5511999999999@s.whatsapp.net\", \"text\": \"Olá!\"}'"
	@echo ""
	@echo "$(GREEN)📚 Documentação completa: http://localhost:8080/swagger/index.html$(NC)"
