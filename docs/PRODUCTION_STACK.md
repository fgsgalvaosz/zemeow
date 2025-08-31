# 🚀 ZeMeow - Stack de Produção com Traefik

Este documento descreve como fazer o deploy da stack ZeMeow em produção usando Docker Swarm e Traefik.

## 📋 Pré-requisitos

### Infraestrutura
- **Docker Swarm** inicializado
- **Traefik** configurado como proxy reverso
- **Portainer** para gerenciamento (opcional, mas recomendado)
- **Domínios** configurados no DNS

### Rede e Volumes
```bash
# Criar rede externa (se não existir)
docker network create --driver overlay Gacont

# Criar volumes necessários
make stack-volumes
```

## 🌐 Domínios Configurados

A stack está configurada para os seguintes domínios (ajuste conforme necessário):

| Serviço | Domínio | Porta Interna | Descrição |
|---------|---------|---------------|-----------|
| **ZeMeow API** | `zemeow.gacont.com.br` | 8080 | API principal e Swagger |
| **MinIO S3** | `zs3.gacont.com.br` | 9000 | API S3 para armazenamento |
| **MinIO Console** | `zminio.gacont.com.br` | 9001 | Interface web MinIO |

## 🔧 Configuração

### Variáveis de Ambiente

As seguintes variáveis estão configuradas no `docker-compose.prod.yml`:

#### Banco de Dados
```yaml
- DB_HOST=postgres
- DB_PORT=5432
- DB_NAME=zemeow
- DB_USER=zemeow
- DB_PASSWORD=zemeow123
- DB_SSL_MODE=disable
```

#### Redis
```yaml
- REDIS_HOST=redis
- REDIS_PORT=6379
- REDIS_PASSWORD=redis123
```

#### MinIO
```yaml
- MINIO_ENDPOINT=minio:9000
- MINIO_ACCESS_KEY=Gacont
- MINIO_SECRET_KEY=WIPcLhjcBoslmOd
- MINIO_USE_SSL=false
- MINIO_BUCKET_NAME=zemeow-media
```

#### API
```yaml
- ADMIN_API_KEY=test123
- PORT=8080
```

## 🚀 Deploy no Portainer

### Método 1: Via Interface Web

1. **Acesse Portainer**
   - Vá para `Stacks > Add Stack`

2. **Configure a Stack**
   - Nome: `zemeow-production`
   - Método: `Web editor`

3. **Cole o Conteúdo**
   - Copie todo o conteúdo de `docker-compose.prod.yml`
   - Ajuste os domínios conforme necessário

4. **Deploy**
   - Clique em `Deploy the stack`

### Método 2: Via Git Repository

1. **Configure Repository**
   - URL: `https://github.com/seu-usuario/zemeow`
   - Arquivo: `docker-compose.prod.yml`

2. **Deploy Automático**
   - Configure webhook para deploy automático

## 🔒 Labels Traefik

### ZeMeow API
```yaml
- traefik.enable=true
- traefik.http.routers.zemeow_api.rule=Host(`zemeow.gacont.com.br`)
- traefik.http.routers.zemeow_api.entrypoints=websecure
- traefik.http.routers.zemeow_api.tls.certresolver=letsencryptresolver
- traefik.http.services.zemeow_api.loadbalancer.server.port=8080
- traefik.http.services.zemeow_api.loadbalancer.passHostHeader=true
- traefik.http.routers.zemeow_api.service=zemeow_api
```

### MinIO S3 API
```yaml
- traefik.http.routers.minio_s3.rule=Host(`s3.gacont.com.br`)
- traefik.http.routers.minio_s3.entrypoints=websecure
- traefik.http.routers.minio_s3.tls.certresolver=letsencryptresolver
- traefik.http.services.minio_s3.loadbalancer.server.port=9000
```

### MinIO Console
```yaml
- traefik.http.routers.minio_console.rule=Host(`minio.gacont.com.br`)
- traefik.http.routers.minio_console.entrypoints=websecure
- traefik.http.routers.minio_console.tls.certresolver=letsencryptresolver
- traefik.http.services.minio_console.loadbalancer.server.port=9001
```

## 📦 Volumes Externos

A stack usa volumes externos para persistência:

```yaml
volumes:
  postgres_data:
    external: true
    name: zemeow_postgres_data
  
  redis_data:
    external: true
    name: zemeow_redis_data
  
  minio_data:
    external: true
    name: zemeow_minio_data
  
  zemeow_sessions:
    external: true
    name: zemeow_sessions_data
  
  zemeow_logs:
    external: true
    name: zemeow_logs_data
```

## 🔍 Monitoramento

### Health Checks
Todos os serviços possuem health checks configurados:
- **PostgreSQL**: Verificação de conexão
- **Redis**: Teste de ping
- **MinIO**: Verificação da API
- **ZeMeow**: Health endpoint `/health`

### Logs
```bash
# Ver logs da stack
docker service logs zemeow-production_zemeow

# Logs em tempo real
docker service logs -f zemeow-production_zemeow
```

## 🛠️ Manutenção

### Atualização da Imagem
```bash
# Atualizar para nova versão
docker service update --image felipyfgs17/zemeow:v1.1.0 zemeow-production_zemeow
```

### Backup
```bash
# Backup PostgreSQL
docker exec -t $(docker ps -q -f name=zemeow-production_postgres) pg_dump -U zemeow zemeow > backup.sql

# Backup MinIO
mc mirror minio/zemeow-media ./backup-minio/
```

### Scaling
```bash
# Escalar ZeMeow API
docker service scale zemeow-production_zemeow=3
```

## 🔧 Troubleshooting

### Problemas Comuns

1. **Serviço não inicia**
   - Verificar logs: `docker service logs zemeow-production_zemeow`
   - Verificar recursos: `docker node ls`

2. **Domínio não resolve**
   - Verificar DNS
   - Verificar labels Traefik
   - Verificar certificados SSL

3. **Banco não conecta**
   - Verificar se PostgreSQL está rodando
   - Verificar credenciais
   - Verificar rede

### Comandos Úteis
```bash
# Status dos serviços
docker service ls

# Detalhes do serviço
docker service inspect zemeow-production_zemeow

# Logs específicos
docker service logs --tail 100 zemeow-production_zemeow

# Reiniciar serviço
docker service update --force zemeow-production_zemeow
```

## 📚 Recursos Adicionais

- **Swagger UI**: `https://zemeow.gacont.com.br/swagger/index.html`
- **MinIO Console**: `https://minio.gacont.com.br`
- **Health Check**: `https://zemeow.gacont.com.br/health`

## 🔐 Segurança

### Recomendações
1. **Alterar senhas padrão** antes do deploy
2. **Configurar firewall** adequadamente
3. **Usar certificados SSL** válidos
4. **Monitorar logs** regularmente
5. **Fazer backups** periódicos

### Variáveis Sensíveis
Considere usar Docker Secrets para:
- `DB_PASSWORD`
- `REDIS_PASSWORD`
- `MINIO_SECRET_KEY`
- `ADMIN_API_KEY`
