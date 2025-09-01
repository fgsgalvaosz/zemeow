# Webhook com Payload Bruto - Exemplos de Uso

Este documento apresenta exemplos práticos de como usar webhooks com payload bruto da WhatsmeOw no sistema ZeMeow.

## Configuração de Webhook com Payload Bruto

### 1. Configurar Webhook em Modo "raw"

```bash
curl -X POST http://localhost:8080/webhooks/sessions/sessionabc123/set \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer SEU_API_KEY" \
  -d '{
    "url": "https://seu-servidor.com/webhook-raw",
    "events": ["message", "receipt", "connected"],
    "payload_mode": "raw",
    "active": true
  }'
```

### 2. Configurar Webhook em Modo "both" (Processado + Bruto)

```bash
curl -X POST http://localhost:8080/webhooks/sessions/sessionabc123/set \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer SEU_API_KEY" \
  -d '{
    "url": "https://seu-servidor.com/webhook-both",
    "events": ["message"],
    "payload_mode": "both",
    "active": true
  }'
```

## Estruturas de Payload

### Payload Processado (Modo "processed" - Padrão)

```json
{
  "session_id": "sessionabc123",
  "event": "message",
  "data": {
    "session_id": "sessionabc123",
    "message_id": "3EB0C7675F85F44B1F",
    "from": "5511987654321@s.whatsapp.net",
    "chat": "5511987654321@s.whatsapp.net",
    "timestamp": 1698408300,
    "message": {
      "conversation": "Olá, como você está?"
    }
  },
  "timestamp": "2024-01-15T14:30:01Z",
  "metadata": {
    "session_name": "Minha Sessão",
    "jid": "5511999999999@s.whatsapp.net",
    "payload_type": "processed"
  }
}
```

### Payload Bruto (Modo "raw")

```json
{
  "session_id": "sessionabc123",
  "event_type": "*events.Message",
  "raw_data": {
    "Info": {
      "MessageSource": {
        "Chat": "5511987654321@s.whatsapp.net",
        "Sender": "5511987654321@s.whatsapp.net",
        "IsFromMe": false,
        "IsGroup": false
      },
      "ID": "3EB0C7675F85F44B1F",
      "Type": "text",
      "PushName": "João Silva",
      "Timestamp": "2024-01-15T14:30:00Z",
      "Category": "",
      "Multicast": false,
      "MediaType": "",
      "Edit": {
        "EditedTimestamp": null,
        "EditedMessageID": ""
      }
    },
    "Message": {
      "conversation": "Olá, como você está?"
    },
    "RawMessage": {
      "key": {
        "remoteJid": "5511987654321@s.whatsapp.net",
        "fromMe": false,
        "id": "3EB0C7675F85F44B1F"
      },
      "messageTimestamp": 1705322600,
      "pushName": "João Silva",
      "message": {
        "conversation": "Olá, como você está?"
      }
    }
  },
  "event_meta": {
    "whatsmeow_version": "v0.0.0-20250611130243",
    "protocol_version": "2.24.6",
    "session_jid": "5511999999999@s.whatsapp.net",
    "device_info": "ZeMeow/1.0",
    "go_version": "go1.24.0"
  },
  "timestamp": "2024-01-15T14:30:01Z",
  "payload_type": "raw"
}
```

## Exemplos de Tipos de Eventos

### 1. Evento de Mensagem (*events.Message)

**Headers HTTP Recebidos:**
```
Content-Type: application/json
User-Agent: ZeMeow-Webhook/1.0
X-Webhook-Event: *events.Message
X-Session-ID: sessionabc123
X-Payload-Type: raw
X-Event-Type: *events.Message
```

**Payload Bruto de Mensagem de Texto:**
```json
{
  "session_id": "sessionabc123",
  "event_type": "*events.Message",
  "raw_data": {
    "Info": {
      "MessageSource": {
        "Chat": "5511987654321@s.whatsapp.net",
        "Sender": "5511987654321@s.whatsapp.net",
        "IsFromMe": false,
        "IsGroup": false
      },
      "ID": "3EB0C7675F85F44B1F",
      "Type": "text",
      "PushName": "João Silva",
      "Timestamp": "2024-01-15T14:30:00Z"
    },
    "Message": {
      "conversation": "Olá! Como você está?"
    }
  },
  "payload_type": "raw"
}
```

**Payload Bruto de Mensagem de Mídia:**
```json
{
  "session_id": "sessionabc123",
  "event_type": "*events.Message",
  "raw_data": {
    "Info": {
      "MessageSource": {
        "Chat": "5511987654321@s.whatsapp.net",
        "Sender": "5511987654321@s.whatsapp.net",
        "IsFromMe": false,
        "IsGroup": false
      },
      "ID": "4FC1D8786A96F55C2G",
      "Type": "image",
      "MediaType": "image"
    },
    "Message": {
      "imageMessage": {
        "url": "https://mmg.whatsapp.net/o1/v/t62.7118-24/...",
        "mimetype": "image/jpeg",
        "caption": "Olha essa foto!",
        "fileSha256": "...",
        "fileLength": 245760,
        "height": 1200,
        "width": 800,
        "mediaKey": "...",
        "fileEncSha256": "...",
        "directPath": "/v/t62.7118-24/..."
      }
    }
  },
  "payload_type": "raw"
}
```

### 2. Evento de Confirmação (*events.Receipt)

```json
{
  "session_id": "sessionabc123",
  "event_type": "*events.Receipt",
  "raw_data": {
    "MessageSource": {
      "Chat": "5511987654321@s.whatsapp.net",
      "Sender": "5511987654321@s.whatsapp.net",
      "IsFromMe": false,
      "IsGroup": false
    },
    "MessageIDs": ["3EB0C7675F85F44B1F"],
    "Timestamp": "2024-01-15T14:30:05Z",
    "Type": "read"
  },
  "payload_type": "raw"
}
```

### 3. Evento de Conexão (*events.Connected)

```json
{
  "session_id": "sessionabc123",
  "event_type": "*events.Connected",
  "raw_data": {},
  "event_meta": {
    "whatsmeow_version": "v0.0.0-20250611130243",
    "session_jid": "5511999999999@s.whatsapp.net"
  },
  "payload_type": "raw"
}
```

## Implementação do Servidor Webhook

### Exemplo em Node.js/Express

```javascript
const express = require('express');
const app = express();

app.use(express.json());

// Endpoint para receber webhooks brutos
app.post('/webhook-raw', (req, res) => {
  const payload = req.body;
  
  console.log('Evento recebido:', payload.event_type);
  console.log('Sessão:', payload.session_id);
  console.log('Headers:', req.headers);
  
  // Processar conforme o tipo de evento
  switch (payload.event_type) {
    case '*events.Message':
      handleRawMessage(payload.raw_data);
      break;
      
    case '*events.Receipt':
      handleRawReceipt(payload.raw_data);
      break;
      
    case '*events.Connected':
      handleConnection(payload);
      break;
      
    default:
      console.log('Evento não tratado:', payload.event_type);
  }
  
  // Sempre responder com 200 para confirmar recebimento
  res.status(200).json({ received: true });
});

function handleRawMessage(rawData) {
  const messageInfo = rawData.Info;
  const message = rawData.Message;
  
  console.log('Mensagem de:', messageInfo.MessageSource.Sender);
  console.log('Chat:', messageInfo.MessageSource.Chat);
  console.log('ID:', messageInfo.ID);
  console.log('Tipo:', messageInfo.Type);
  console.log('Conteúdo:', message);
  
  // Acessar dados específicos que não estavam disponíveis no payload processado
  if (messageInfo.Edit && messageInfo.Edit.EditedTimestamp) {
    console.log('Mensagem editada em:', messageInfo.Edit.EditedTimestamp);
  }
  
  if (messageInfo.Category) {
    console.log('Categoria:', messageInfo.Category);
  }
  
  // Processar diferentes tipos de mensagem
  if (message.conversation) {
    console.log('Texto:', message.conversation);
  } else if (message.imageMessage) {
    console.log('Imagem recebida');
    console.log('Caption:', message.imageMessage.caption);
    console.log('Mime:', message.imageMessage.mimetype);
    console.log('Tamanho:', message.imageMessage.fileLength);
  }
}

function handleRawReceipt(rawData) {
  console.log('Confirmação de leitura para mensagens:', rawData.MessageIDs);
  console.log('Tipo:', rawData.Type);
  console.log('Chat:', rawData.MessageSource.Chat);
}

function handleConnection(payload) {
  console.log('Sessão conectada:', payload.session_id);
  console.log('JID:', payload.event_meta.session_jid);
  console.log('Versão WhatsmeOw:', payload.event_meta.whatsmeow_version);
}

app.listen(3000, () => {
  console.log('Servidor webhook executando na porta 3000');
});
```

### Exemplo em Python/Flask

```python
from flask import Flask, request, jsonify
import json

app = Flask(__name__)

@app.route('/webhook-raw', methods=['POST'])
def webhook_raw():
    payload = request.get_json()
    
    print(f"Evento recebido: {payload.get('event_type')}")
    print(f"Sessão: {payload.get('session_id')}")
    print(f"Headers: {dict(request.headers)}")
    
    event_type = payload.get('event_type')
    raw_data = payload.get('raw_data', {})
    
    if event_type == '*events.Message':
        handle_raw_message(raw_data)
    elif event_type == '*events.Receipt':
        handle_raw_receipt(raw_data)
    elif event_type == '*events.Connected':
        handle_connection(payload)
    else:
        print(f"Evento não tratado: {event_type}")
    
    return jsonify({"received": True}), 200

def handle_raw_message(raw_data):
    message_info = raw_data.get('Info', {})
    message = raw_data.get('Message', {})
    
    sender = message_info.get('MessageSource', {}).get('Sender')
    chat = message_info.get('MessageSource', {}).get('Chat')
    message_id = message_info.get('ID')
    message_type = message_info.get('Type')
    
    print(f"Mensagem de: {sender}")
    print(f"Chat: {chat}")
    print(f"ID: {message_id}")
    print(f"Tipo: {message_type}")
    
    # Acessar dados específicos não disponíveis no payload processado
    edit_info = message_info.get('Edit', {})
    if edit_info.get('EditedTimestamp'):
        print(f"Mensagem editada em: {edit_info['EditedTimestamp']}")
    
    category = message_info.get('Category')
    if category:
        print(f"Categoria: {category}")
    
    # Processar diferentes tipos de mensagem
    if 'conversation' in message:
        print(f"Texto: {message['conversation']}")
    elif 'imageMessage' in message:
        image_msg = message['imageMessage']
        print("Imagem recebida")
        print(f"Caption: {image_msg.get('caption', '')}")
        print(f"Mime: {image_msg.get('mimetype')}")
        print(f"Tamanho: {image_msg.get('fileLength')}")

def handle_raw_receipt(raw_data):
    message_ids = raw_data.get('MessageIDs', [])
    receipt_type = raw_data.get('Type')
    chat = raw_data.get('MessageSource', {}).get('Chat')
    
    print(f"Confirmação de {receipt_type} para mensagens: {message_ids}")
    print(f"Chat: {chat}")

def handle_connection(payload):
    session_id = payload.get('session_id')
    event_meta = payload.get('event_meta', {})
    
    print(f"Sessão conectada: {session_id}")
    print(f"JID: {event_meta.get('session_jid')}")
    print(f"Versão WhatsmeOw: {event_meta.get('whatsmeow_version')}")

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=3000, debug=True)
```

## Vantagens do Payload Bruto

### 1. **Dados Completos**
- Acesso a todos os campos da estrutura original da WhatsmeOw
- Metadados detalhados que são perdidos na serialização processada
- Informações de edição, categoria, multicast, etc.

### 2. **Compatibilidade Futura**
- Novos campos da WhatsmeOw são automaticamente incluídos
- Sem necessidade de atualizações no sistema ZeMeow
- Mantém compatibilidade com mudanças no protocolo WhatsApp

### 3. **Flexibilidade**
- Processamento personalizado no lado do cliente
- Acesso a estruturas complexas como `RawMessage`
- Controle total sobre como os dados são interpretados

### 4. **Depuração**
- Facilita debugging de problemas
- Visibilidade completa dos dados originais
- Identificação de novos tipos de eventos

## Considerações de Performance

### 1. **Tamanho do Payload**
- Payloads brutos são maiores que processados
- Use modo "raw" apenas quando necessário
- Considere modo "both" apenas para debugging

### 2. **Processamento**
- Payloads brutos requerem mais processamento no cliente
- Estruturas podem ser complexas e aninhadas
- Implemente caching se necessário

### 3. **Largura de Banda**
- Monitorar uso de largura de banda
- Filtrar eventos apenas para os necessários
- Considere compressão HTTP

## Migração de Payload Processado para Bruto

### Passo 1: Configurar Webhook em Modo "both"
```bash
curl -X POST http://localhost:8080/webhooks/sessions/sessionabc123/set \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer SEU_API_KEY" \
  -d '{
    "url": "https://seu-servidor.com/webhook-migration",
    "events": ["message"],
    "payload_mode": "both",
    "active": true
  }'
```

### Passo 2: Implementar Handler Duplo
```javascript
app.post('/webhook-migration', (req, res) => {
  const payload = req.body;
  
  // Detectar tipo de payload pelos headers
  const payloadType = req.headers['x-payload-type'];
  
  if (payloadType === 'raw') {
    // Processar payload bruto
    handleRawPayload(payload);
  } else {
    // Processar payload processado (legado)
    handleProcessedPayload(payload);
  }
  
  res.status(200).json({ received: true });
});
```

### Passo 3: Mudar para Modo "raw"
Após validar que o processamento bruto está funcionando:
```bash
curl -X POST http://localhost:8080/webhooks/sessions/sessionabc123/set \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer SEU_API_KEY" \
  -d '{
    "url": "https://seu-servidor.com/webhook-raw",
    "events": ["message"],
    "payload_mode": "raw",
    "active": true
  }'
```

## Troubleshooting

### Problema: Payloads vazios ou nulos
**Solução:** Verificar se a sessão está configurada corretamente e os eventos estão habilitados.

### Problema: Estruturas diferentes do esperado
**Solução:** A WhatsmeOw pode alterar estruturas. Sempre validar campos antes de acessar.

### Problema: Performance degradada
**Solução:** Filtrar eventos desnecessários e otimizar processamento no cliente.

### Problema: Headers não recebidos
**Solução:** Verificar se o servidor webhook está processando headers HTTP corretamente.

## Suporte e Recursos Adicionais

- **Documentação WhatsmeOw:** https://pkg.go.dev/go.mau.fi/whatsmeow
- **Logs do Sistema:** Verificar logs do ZeMeow para debugging
- **API Reference:** /api endpoint para documentação Swagger
- **Health Check:** /health endpoint para status do sistema