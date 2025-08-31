#!/bin/bash

# Script para testar todos os endpoints da API ZeMeow
# Configurações
BASE_URL="http://localhost:8080"
API_KEY="test123"
PHONE_NUMBER="559984059035"
SESSION_ID=""

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Função para log
log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Função para fazer requisições
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
    
    # Separar body e status code
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

# 1. Health Check
log "=== HEALTH CHECK ==="
make_request "GET" "/health" "" "Health Check"

# 2. Verificar sessões existentes
log "=== VERIFICANDO SESSÕES EXISTENTES ==="
response=$(curl -s -X GET \
    -H "X-API-Key: $API_KEY" \
    "$BASE_URL/sessions")

echo "Response: $response"

# Extrair session ID da primeira sessão disponível
SESSION_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$SESSION_ID" ]; then
    success "Usando sessão existente com ID: $SESSION_ID"
else
    log "Tentando criar nova sessão..."
    session_data='{"name": "Test Session Auto"}'

    create_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "X-API-Key: $API_KEY" \
        -d "$session_data" \
        "$BASE_URL/sessions/add")

    echo "Create Response: $create_response"
    SESSION_ID=$(echo "$create_response" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

    if [ -z "$SESSION_ID" ]; then
        error "Falha ao criar sessão"
        exit 1
    fi
    success "Nova sessão criada com ID: $SESSION_ID"
fi

# 3. Listar Sessões
log "=== LISTANDO SESSÕES ==="
make_request "GET" "/sessions" "" "Listar todas as sessões"

# 4. Obter detalhes da sessão
log "=== DETALHES DA SESSÃO ==="
make_request "GET" "/sessions/$SESSION_ID" "" "Obter detalhes da sessão"

# 5. Status da sessão
log "=== STATUS DA SESSÃO ==="
make_request "GET" "/sessions/$SESSION_ID/status" "" "Status da sessão"

# 6. QR Code (se necessário)
log "=== QR CODE ==="
make_request "GET" "/sessions/$SESSION_ID/qr" "" "Obter QR Code"

# Aguardar um pouco para possível conexão
log "Aguardando 5 segundos para possível conexão..."
sleep 5

# 7. TESTES DE ENVIO DE MENSAGENS
log "=== TESTES DE ENVIO ==="

# 7.1 Enviar texto
text_data='{
  "to": "'"$PHONE_NUMBER"'",
  "text": "🤖 Teste automático da API ZeMeow - Mensagem de texto simples",
  "message_id": "test_text_'"$(date +%s)"'"
}'
make_request "POST" "/sessions/$SESSION_ID/send/text" "$text_data" "Enviar mensagem de texto"

# 7.2 Enviar mídia (imagem pequena)
media_data='{
  "to": "'"$PHONE_NUMBER"'",
  "type": "image",
  "media": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
  "caption": "🖼️ Teste de imagem da API ZeMeow"
}'
make_request "POST" "/sessions/$SESSION_ID/send/media" "$media_data" "Enviar mídia (imagem)"

# 7.3 Enviar localização
location_data='{
  "to": "'"$PHONE_NUMBER"'",
  "latitude": -23.550520,
  "longitude": -46.633308,
  "name": "📍 São Paulo - Teste API ZeMeow"
}'
make_request "POST" "/sessions/$SESSION_ID/send/location" "$location_data" "Enviar localização"

# 7.4 Enviar contato
contact_data='{
  "to": "'"$PHONE_NUMBER"'",
  "name": "👤 Contato Teste ZeMeow",
  "vcard": "BEGIN:VCARD\nVERSION:3.0\nFN:Contato Teste ZeMeow\nTEL;TYPE=CELL:+5511999999999\nEND:VCARD"
}'
make_request "POST" "/sessions/$SESSION_ID/send/contact" "$contact_data" "Enviar contato"

# 7.5 Enviar sticker
sticker_data='{
  "to": "'"$PHONE_NUMBER"'",
  "sticker": "data:image/webp;base64,UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA",
  "message_id": "test_sticker_'"$(date +%s)"'"
}'
make_request "POST" "/sessions/$SESSION_ID/send/sticker" "$sticker_data" "Enviar sticker"

# 7.6 Enviar botões
buttons_data='{
  "to": "'"$PHONE_NUMBER"'",
  "text": "🔘 Escolha uma opção nos botões abaixo:",
  "buttons": [
    {
      "id": "btn1",
      "text": "✅ Opção 1"
    },
    {
      "id": "btn2", 
      "text": "❌ Opção 2"
    }
  ],
  "footer": "🤖 Powered by ZeMeow API"
}'
make_request "POST" "/sessions/$SESSION_ID/send/buttons" "$buttons_data" "Enviar botões interativos"

# 7.7 Enviar lista
list_data='{
  "to": "'"$PHONE_NUMBER"'",
  "text": "📋 Selecione um item da lista:",
  "button_text": "Ver Opções",
  "sections": [
    {
      "title": "🔧 Seção Teste",
      "rows": [
        {
          "id": "item1",
          "title": "📱 Item 1",
          "description": "Descrição do primeiro item"
        },
        {
          "id": "item2",
          "title": "💻 Item 2", 
          "description": "Descrição do segundo item"
        }
      ]
    }
  ]
}'
make_request "POST" "/sessions/$SESSION_ID/send/list" "$list_data" "Enviar lista interativa"

# 7.8 Enviar enquete
poll_data='{
  "to": "'"$PHONE_NUMBER"'",
  "name": "🗳️ Qual sua cor favorita?",
  "options": ["🔵 Azul", "🟢 Verde", "🔴 Vermelho", "🟡 Amarelo"],
  "selectable_count": 1
}'
make_request "POST" "/sessions/$SESSION_ID/send/poll" "$poll_data" "Enviar enquete"

echo ""
log "=== TESTES DE CHAT ==="

# 8. Presença no chat
presence_data='{
  "to": "'"$PHONE_NUMBER"'",
  "presence": "composing"
}'
make_request "POST" "/sessions/$SESSION_ID/presence" "$presence_data" "Definir presença (digitando)"

# Aguardar um pouco
sleep 2

# Parar de digitar
presence_stop_data='{
  "to": "'"$PHONE_NUMBER"'",
  "presence": "paused"
}'
make_request "POST" "/sessions/$SESSION_ID/presence" "$presence_stop_data" "Parar presença (parou de digitar)"

echo ""
log "=== TESTES DE GRUPOS ==="

# 9. Criar grupo
group_data='{
  "name": "🤖 Grupo Teste ZeMeow API",
  "participants": ["'"$PHONE_NUMBER"'"],
  "description": "Grupo criado automaticamente para testes da API ZeMeow"
}'
make_request "POST" "/sessions/$SESSION_ID/group/create" "$group_data" "Criar grupo"

# 10. Listar grupos
make_request "GET" "/sessions/$SESSION_ID/group/list" "" "Listar grupos"

echo ""
log "=== TESTES DE INFORMAÇÕES ==="

# 11. Informações de contato
info_data='{
  "phones": ["'"$PHONE_NUMBER"'"]
}'
make_request "POST" "/sessions/$SESSION_ID/check" "$info_data" "Verificar contatos no WhatsApp"

# 12. Avatar de contato
avatar_data='{
  "phone": "'"$PHONE_NUMBER"'"
}'
make_request "POST" "/sessions/$SESSION_ID/avatar" "$avatar_data" "Obter avatar de contato"

# 13. Listar contatos
make_request "GET" "/sessions/$SESSION_ID/contacts" "" "Listar contatos"

# 14. Listar newsletters
make_request "GET" "/sessions/$SESSION_ID/newsletter/list" "" "Listar newsletters"

echo ""
log "=== ESTATÍSTICAS FINAIS ==="
make_request "GET" "/sessions/$SESSION_ID/stats" "" "Estatísticas da sessão"

echo ""
success "🎉 Teste completo finalizado!"
warning "📝 Verifique o número $PHONE_NUMBER para confirmar o recebimento das mensagens"
log "🔧 Session ID usado: $SESSION_ID"
