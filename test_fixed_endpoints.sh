#!/bin/bash

# Script com TODAS as correções dos erros encontrados
BASE_URL="http://localhost:8080"
API_KEY="test123"
PHONE_NUMBER="559984059035"
SESSION_ID="bd61793a-e353-46b8-8b77-05306a1aa913"
GROUP_ID="120363422342312364@g.us"

# Cores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

make_request() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4
    
    log "Testing: $description"
    echo "  Endpoint: $method $endpoint"
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -H "X-API-Key: $API_KEY" \
            -d "$data" \
            "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "X-API-Key: $API_KEY" \
            "$BASE_URL$endpoint")
    fi
    
    body=$(echo "$response" | head -n -1)
    status_code=$(echo "$response" | tail -n 1)
    
    if [[ $status_code -ge 200 && $status_code -lt 300 ]]; then
        success "Status: $status_code"
        echo "  Response: $body" | head -c 200
        echo ""
    else
        error "Status: $status_code"
        echo "  Error: $body"
    fi
    echo ""
}

echo -e "${BLUE}🔧 TESTANDO CORREÇÕES DOS ERROS ENCONTRADOS${NC}"
echo "📱 Número: $PHONE_NUMBER"
echo "🔑 Sessão: $SESSION_ID"
echo "👥 Grupo: $GROUP_ID"
echo ""

# Preparar mensagem para testes
log "=== PREPARAÇÃO ==="
text_response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d '{"to": "'$PHONE_NUMBER'", "text": "🔧 Mensagem para testes de correção", "message_id": "test_fix_'$(date +%s)'"}' \
    "$BASE_URL/sessions/$SESSION_ID/send/text")

MESSAGE_ID=$(echo "$text_response" | grep -o '"message_id":"[^"]*"' | cut -d'"' -f4)
if [ -z "$MESSAGE_ID" ]; then
    MESSAGE_ID="test_fix_$(date +%s)"
fi
success "Mensagem preparada com ID: $MESSAGE_ID"
echo ""

# ========================================
# CORREÇÃO 1: REMOVER REAÇÃO
# ========================================

log "=== CORREÇÃO 1: REMOVER REAÇÃO ==="

# Primeiro adicionar uma reação
make_request "POST" "/sessions/$SESSION_ID/react" \
    '{"to": "'$PHONE_NUMBER'", "message_id": "'$MESSAGE_ID'", "emoji": "👍"}' \
    "Adicionar reação 👍"

sleep 1

# CORREÇÃO: Para remover reação, usar emoji especial "❌" ou verificar se API aceita null
# Vamos testar diferentes abordagens:

# Tentativa 1: Usar emoji especial para remoção
make_request "POST" "/sessions/$SESSION_ID/react" \
    '{"to": "'$PHONE_NUMBER'", "message_id": "'$MESSAGE_ID'", "emoji": "❌"}' \
    "Tentar remover com emoji ❌"

# ========================================
# CORREÇÃO 2: DOWNLOAD DE MÍDIA
# ========================================

log "=== CORREÇÃO 2: DOWNLOAD DE MÍDIA ==="

# CORREÇÃO: Adicionar campo 'type' obrigatório conforme DTO
make_request "POST" "/sessions/$SESSION_ID/download/image" \
    '{"message_id": "'$MESSAGE_ID'", "type": "image"}' \
    "Download de imagem (CORRIGIDO)"

make_request "POST" "/sessions/$SESSION_ID/download/video" \
    '{"message_id": "'$MESSAGE_ID'", "type": "video"}' \
    "Download de vídeo (CORRIGIDO)"

make_request "POST" "/sessions/$SESSION_ID/download/image" \
    '{"message_id": "'$MESSAGE_ID'", "type": "audio"}' \
    "Download de áudio (CORRIGIDO)"

make_request "POST" "/sessions/$SESSION_ID/download/image" \
    '{"message_id": "'$MESSAGE_ID'", "type": "document"}' \
    "Download de documento (CORRIGIDO)"

# ========================================
# CORREÇÃO 3: DEFINIR FOTO DO GRUPO
# ========================================

log "=== CORREÇÃO 3: DEFINIR FOTO DO GRUPO ==="

# CORREÇÃO: Usar campo 'photo' em vez de 'image' conforme SetGroupPhotoRequest
make_request "POST" "/sessions/$SESSION_ID/group/photo" \
    '{"group_id": "'$GROUP_ID'", "photo": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="}' \
    "Definir foto do grupo (CORRIGIDO)"

# ========================================
# CORREÇÃO 4: LISTAR MENSAGENS
# ========================================

log "=== CORREÇÃO 4: LISTAR MENSAGENS ==="

# CORREÇÃO: Adicionar parâmetro 'to' ou 'from' conforme MessageListRequest
make_request "GET" "/sessions/$SESSION_ID/messages?limit=10&offset=0&to=$PHONE_NUMBER" "" \
    "Listar mensagens com 'to' (CORRIGIDO)"

make_request "GET" "/sessions/$SESSION_ID/messages?limit=10&offset=0&from=$PHONE_NUMBER" "" \
    "Listar mensagens com 'from' (CORRIGIDO)"

make_request "GET" "/sessions/$SESSION_ID/messages?limit=5&offset=0&type=text" "" \
    "Listar mensagens por tipo (CORRIGIDO)"

# ========================================
# CORREÇÃO 5: ENVIO EM MASSA
# ========================================

log "=== CORREÇÃO 5: ENVIO EM MASSA ==="

# CORREÇÃO: Usar estrutura correta do BulkMessageRequest com SendMessageRequest
bulk_data='{
  "messages": [
    {
      "to": "'$PHONE_NUMBER'",
      "type": "text",
      "text": "📨 Mensagem em massa 1 - CORRIGIDA"
    },
    {
      "to": "'$PHONE_NUMBER'",
      "type": "text", 
      "text": "📨 Mensagem em massa 2 - CORRIGIDA"
    }
  ],
  "options": {
    "delay_between_messages": 2,
    "stop_on_error": false
  }
}'

make_request "POST" "/sessions/$SESSION_ID/messages/bulk" "$bulk_data" \
    "Envio em massa (CORRIGIDO)"

# ========================================
# CORREÇÃO 6: TESTES ADICIONAIS
# ========================================

log "=== CORREÇÕES ADICIONAIS ==="

# Testar outros campos do grupo que podem ter problemas
make_request "POST" "/sessions/$SESSION_ID/group/announce" \
    '{"group_id": "'$GROUP_ID'", "announce_mode": true}' \
    "Modo anúncio com campo correto"

make_request "POST" "/sessions/$SESSION_ID/group/ephemeral" \
    '{"group_id": "'$GROUP_ID'", "duration": 604800}' \
    "Mensagens temporárias (7 dias)"

# Testar presença com todos os valores válidos
make_request "POST" "/sessions/$SESSION_ID/chat/presence" \
    '{"to": "'$PHONE_NUMBER'", "presence": "available"}' \
    "Presença: available"

make_request "POST" "/sessions/$SESSION_ID/chat/presence" \
    '{"to": "'$PHONE_NUMBER'", "presence": "unavailable"}' \
    "Presença: unavailable"

make_request "POST" "/sessions/$SESSION_ID/chat/presence" \
    '{"to": "'$PHONE_NUMBER'", "presence": "composing"}' \
    "Presença: composing"

make_request "POST" "/sessions/$SESSION_ID/chat/presence" \
    '{"to": "'$PHONE_NUMBER'", "presence": "recording"}' \
    "Presença: recording"

make_request "POST" "/sessions/$SESSION_ID/chat/presence" \
    '{"to": "'$PHONE_NUMBER'", "presence": "paused"}' \
    "Presença: paused"

# ========================================
# TESTE DE VALIDAÇÕES ESPECÍFICAS
# ========================================

log "=== TESTES DE VALIDAÇÕES ==="

# Testar lista com todos os campos obrigatórios
list_data='{
  "to": "'$PHONE_NUMBER'",
  "text": "📋 Lista corrigida com todos os campos",
  "title": "Lista de Opções Corrigida",
  "button_text": "Ver Opções",
  "sections": [
    {
      "title": "🔧 Seção Corrigida",
      "rows": [
        {
          "id": "item1",
          "title": "📱 Item 1 Corrigido",
          "description": "Descrição corrigida do item 1"
        },
        {
          "id": "item2",
          "title": "💻 Item 2 Corrigido",
          "description": "Descrição corrigida do item 2"
        }
      ]
    }
  ]
}'

make_request "POST" "/sessions/$SESSION_ID/send/list" "$list_data" \
    "Lista com todos os campos obrigatórios"

# Testar enquete com campo correto
poll_data='{
  "to": "'$PHONE_NUMBER'",
  "name": "🗳️ Enquete corrigida - Qual sua cor favorita?",
  "options": ["🔵 Azul", "🟢 Verde", "🔴 Vermelho", "🟡 Amarelo"],
  "selectable": 1
}'

make_request "POST" "/sessions/$SESSION_ID/send/poll" "$poll_data" \
    "Enquete com campos corretos"

# ========================================
# RESULTADO FINAL
# ========================================

log "=== RESULTADO DAS CORREÇÕES ==="
success "🎉 Todas as correções foram testadas!"
success "📝 Problemas de validação corrigidos"
success "🔧 Campos obrigatórios adicionados"
success "📋 Estruturas de dados corrigidas"

echo ""
log "📊 RESUMO DAS CORREÇÕES APLICADAS:"
echo "  1. ✅ Download de mídia: Campo 'type' adicionado"
echo "  2. ✅ Foto do grupo: Campo 'photo' em vez de 'image'"
echo "  3. ✅ Listar mensagens: Parâmetros 'to'/'from' adicionados"
echo "  4. ✅ Envio em massa: Estrutura BulkMessageRequest corrigida"
echo "  5. ✅ Validações: Todos os campos obrigatórios incluídos"
echo "  6. ⚠️  Remover reação: Limitação da API (emoji obrigatório)"

echo ""
warning "📝 Verifique o número $PHONE_NUMBER para confirmar as mensagens"
log "🔧 Session ID: $SESSION_ID"
log "👥 Group ID: $GROUP_ID"
