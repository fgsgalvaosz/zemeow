# 🔗 Testando Webhooks com Webhook Tester

Este guia explica como usar o **Webhook Tester** integrado ao ZeMeow para testar webhooks de forma fácil e visual.

## 🚀 Iniciando o Webhook Tester

### 1. Iniciar o serviço via Docker Compose

```bash
# Iniciar apenas o webhook-tester
docker-compose --profile webhook-test up webhook-tester -d

# Ou iniciar todos os serviços incluindo webhook-tester
docker-compose --profile webhook-test up -d
```

### 2. Acessar a interface web

Abra seu navegador e acesse:
```
http://localhost:8090
```

## 📋 Como Usar

### 1. **Criar uma Sessão de Teste**
- Acesse `http://localhost:8090`
- Clique em **"Create new session"**
- Você receberá uma URL única para testes, exemplo:
  ```
  http://localhost:8090/webhook/abc123def456
  ```

**Nota**: O webhook-tester aceita webhooks e os armazena para visualização, mas sempre retorna uma página HTML. Isso é normal e não indica erro.

### 2. **Configurar Webhook no ZeMeow**
Use a URL gerada para configurar um webhook:

```bash
curl -X POST "http://localhost:8080/webhooks/sessions/YOUR_SESSION_ID/set" \
  -H "apikey: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://localhost:8090/webhook/abc123def456",
    "events": ["message", "receipt", "presence", "connected", "disconnected"],
    "active": true
  }'
```

### 3. **Testar Webhook Manual**
Envie um webhook de teste:

```bash
curl -X POST "http://localhost:8080/webhooks/send" \
  -H "apikey: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://localhost:8090/webhook/abc123def456",
    "method": "POST",
    "payload": {
      "session_id": "test-session",
      "event": "message",
      "data": {
        "from": "5511999999999@s.whatsapp.net",
        "message": "Hello from ZeMeow!",
        "timestamp": 1756642800
      }
    }
  }'
```

### 4. **Visualizar Requisições**
- Volte para `http://localhost:8090`
- Você verá todas as requisições recebidas em tempo real
- Clique em qualquer requisição para ver detalhes completos

## 🔧 Configurações Avançadas

### Variáveis de Ambiente

No arquivo `.env`, você pode configurar:

```env
# Porta do webhook-tester (padrão: 8090)
WEBHOOK_TESTER_PORT=8090
```

### Configurações do Container

O webhook-tester está configurado com:
- **Máximo de requisições por sessão**: 1000
- **TTL da sessão**: 24 horas
- **Driver de armazenamento**: memória (dados perdidos ao reiniciar)

## 📊 Recursos do Webhook Tester

### ✅ **Interface Web Intuitiva**
- Lista todas as requisições recebidas
- Mostra timestamp, método HTTP, headers
- Exibe payload JSON formatado

### ✅ **Detalhes Completos**
- Headers HTTP completos
- Body da requisição
- Query parameters
- Informações de timing

### ✅ **Filtros e Busca**
- Filtrar por método HTTP
- Buscar por conteúdo
- Ordenar por timestamp

### ✅ **Exportação**
- Exportar requisições como JSON
- Copiar cURL commands
- Compartilhar sessões

## 🧪 Cenários de Teste

### 1. **Teste de Conectividade**
```bash
# Testar se o webhook está acessível
curl -X POST "http://localhost:8090/webhook/YOUR_SESSION_ID" \
  -H "Content-Type: application/json" \
  -d '{"test": "connectivity"}'
```

### 2. **Teste de Eventos WhatsApp**
```bash
# Simular evento de mensagem
curl -X POST "http://localhost:8080/webhooks/send" \
  -H "apikey: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://localhost:8090/webhook/YOUR_SESSION_ID",
    "method": "POST",
    "payload": {
      "session_id": "bd61793a-e353-46b8-8b77-05306a1aa913",
      "event": "message",
      "data": {
        "id": "msg_123",
        "from": "5511999999999@s.whatsapp.net",
        "to": "5511888888888@s.whatsapp.net",
        "message": {
          "type": "text",
          "text": "Olá! Esta é uma mensagem de teste."
        },
        "timestamp": 1756642800
      }
    }
  }'
```

### 3. **Teste de Eventos de Conexão**
```bash
# Simular evento de conexão
curl -X POST "http://localhost:8080/webhooks/send" \
  -H "apikey: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "http://localhost:8090/webhook/YOUR_SESSION_ID",
    "method": "POST",
    "payload": {
      "session_id": "bd61793a-e353-46b8-8b77-05306a1aa913",
      "event": "connected",
      "data": {
        "jid": "5511999999999:1@s.whatsapp.net",
        "timestamp": 1756642800
      }
    }
  }'
```

## 🔍 Troubleshooting

### **Webhook Tester não está acessível**
```bash
# Verificar se o container está rodando
docker-compose ps webhook-tester

# Ver logs do container
docker-compose logs webhook-tester

# Reiniciar o serviço
docker-compose restart webhook-tester
```

### **Webhooks não chegam no tester**
1. Verifique se a URL está correta
2. Confirme que o ZeMeow consegue acessar `localhost:8090`
3. Verifique os logs do ZeMeow para erros de webhook

### **Sessão expirou**
- Sessões duram 24 horas por padrão
- Crie uma nova sessão se necessário
- Configure um TTL maior se precisar

## 🎯 Próximos Passos

Após testar com o webhook-tester:
1. **Configure webhooks reais** para seu ambiente de produção
2. **Implemente handlers** para processar os eventos
3. **Configure retry logic** para webhooks que falharem
4. **Monitore logs** para identificar problemas

---

**🎉 Agora você pode testar webhooks do ZeMeow de forma visual e interativa!**
