# ZeMeow WhatsApp API - Dockerfile
# Multi-stage build para otimizar o tamanho da imagem

# Estágio 1: Build
FROM golang:1.24-alpine AS builder

# Instalar dependências necessárias
RUN apk add --no-cache git ca-certificates tzdata

# Definir diretório de trabalho
WORKDIR /app

# Copiar arquivos de dependências
COPY go.mod go.sum ./

# Baixar dependências
RUN go mod download

# Copiar código fonte
COPY . .

# Compilar a aplicação
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o zemeow \
    cmd/zemeow/main.go

# Estágio 2: Runtime
FROM alpine:latest

# Instalar dependências de runtime
RUN apk --no-cache add ca-certificates curl

# Criar usuário não-root
RUN addgroup -g 1001 -S zemeow && \
    adduser -u 1001 -S zemeow -G zemeow

# Definir diretório de trabalho
WORKDIR /app

# Copiar binário do estágio de build
COPY --from=builder /app/zemeow .

# Copiar arquivos de configuração e migrações
COPY --from=builder /app/docs ./docs
COPY --from=builder /app/internal/db/migrations ./internal/db/migrations

# Criar diretórios necessários
RUN mkdir -p /app/sessions /app/logs && \
    chown -R zemeow:zemeow /app

# Mudar para usuário não-root
USER zemeow

# Expor porta
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Comando padrão
CMD ["./zemeow"]
