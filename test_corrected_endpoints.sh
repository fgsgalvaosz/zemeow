#!/bin/bash

# Script corrigido para testar endpoints com problemas identificados
BASE_URL="http://localhost:8080"
API_KEY="test123"
PHONE_NUMBER="559984059035"
SESSION_ID="bd61793a-e353-46b8-8b77-05306a1aa913"

# Cores para output
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

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
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
        echo "  Response: $body" | head -c 300
        echo ""
    else
        error "Status: $status_code"
        echo "  Error: $body"
    fi
    echo ""
}

log "=== TESTES CORRIGIDOS DOS ENDPOINTS COM PROBLEMAS ==="
log "Usando sessão: $SESSION_ID"
log "Número de destino: $PHONE_NUMBER"
echo ""

# 1. Teste de presença corrigido
log "=== TESTE DE PRESENÇA CORRIGIDO ==="
presence_data='{
  "presence": "available"
}'
make_request "POST" "/sessions/$SESSION_ID/presence" "$presence_data" "Definir presença como disponível"

presence_data2='{
  "presence": "unavailable"
}'
make_request "POST" "/sessions/$SESSION_ID/presence" "$presence_data2" "Definir presença como indisponível"

# 2. Teste de verificação de contatos corrigido
log "=== TESTE DE VERIFICAÇÃO DE CONTATOS CORRIGIDO ==="
check_data='{
  "phones": ["'"$PHONE_NUMBER"'"]
}'
make_request "POST" "/sessions/$SESSION_ID/check" "$check_data" "Verificar contatos (campo phones)"

# 3. Teste de lista corrigido com title
log "=== TESTE DE LISTA CORRIGIDO ==="
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
make_request "POST" "/sessions/$SESSION_ID/send/list" "$list_data" "Enviar lista com title corrigido"

# 4. Teste de conectar sessão (para investigar cliente WhatsApp)
log "=== TESTE DE CONEXÃO DA SESSÃO ==="
make_request "POST" "/sessions/$SESSION_ID/connect" "" "Tentar conectar sessão"

# 5. Teste de desconectar e reconectar
log "=== TESTE DE DESCONEXÃO E RECONEXÃO ==="
make_request "POST" "/sessions/$SESSION_ID/disconnect" "" "Desconectar sessão"

sleep 3

make_request "POST" "/sessions/$SESSION_ID/connect" "" "Reconectar sessão"

sleep 5

# 6. Testar novamente alguns endpoints após reconexão
log "=== RETESTANDO ENDPOINTS APÓS RECONEXÃO ==="

# Testar mídia novamente
media_data='{
  "to": "'"$PHONE_NUMBER"'",
  "type": "image",
  "media": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
  "caption": "🖼️ Teste de imagem após reconexão"
}'
make_request "POST" "/sessions/$SESSION_ID/send/media" "$media_data" "Enviar mídia após reconexão"

# Testar localização novamente
location_data='{
  "to": "'"$PHONE_NUMBER"'",
  "latitude": -23.550520,
  "longitude": -46.633308,
  "name": "📍 São Paulo - Teste após reconexão"
}'
make_request "POST" "/sessions/$SESSION_ID/send/location" "$location_data" "Enviar localização após reconexão"

# Testar contato novamente
contact_data='{
  "to": "'"$PHONE_NUMBER"'",
  "name": "👤 Contato Teste Reconexão",
  "vcard": "BEGIN:VCARD\nVERSION:3.0\nFN:Contato Teste Reconexão\nTEL;TYPE=CELL:+5511999999999\nEND:VCARD"
}'
make_request "POST" "/sessions/$SESSION_ID/send/contact" "$contact_data" "Enviar contato após reconexão"

# Testar grupos novamente
group_data='{
  "name": "🤖 Grupo Teste Reconexão",
  "participants": ["'"$PHONE_NUMBER"'"],
  "description": "Grupo criado após reconexão para testes"
}'
make_request "POST" "/sessions/$SESSION_ID/group/create" "$group_data" "Criar grupo após reconexão"

make_request "GET" "/sessions/$SESSION_ID/group/list" "" "Listar grupos após reconexão"

# Testar contatos
make_request "GET" "/sessions/$SESSION_ID/contacts" "" "Listar contatos após reconexão"

# Testar avatar
avatar_data='{
  "phone": "'"$PHONE_NUMBER"'"
}'
make_request "POST" "/sessions/$SESSION_ID/avatar" "$avatar_data" "Obter avatar após reconexão"

# 7. Verificar status final
log "=== STATUS FINAL ==="
make_request "GET" "/sessions/$SESSION_ID/status" "" "Status final da sessão"

log "=== TESTE DE ENDPOINTS ADICIONAIS ==="

# Testar endpoints que não foram testados antes
log "Testando endpoints adicionais..."

# Testar react (reação a mensagem)
react_data='{
  "to": "'"$PHONE_NUMBER"'",
  "message_id": "test_text_1756606281",
  "emoji": "👍"
}'
make_request "POST" "/sessions/$SESSION_ID/react" "$react_data" "Reagir a mensagem"

# Testar marcar como lido
markread_data='{
  "to": "'"$PHONE_NUMBER"'",
  "message_id": ["test_text_1756606281"]
}'
make_request "POST" "/sessions/$SESSION_ID/chat/markread" "$markread_data" "Marcar mensagem como lida"

# Testar editar mensagem
edit_data='{
  "to": "'"$PHONE_NUMBER"'",
  "message_id": "test_text_1756606281",
  "text": "Mensagem editada via API!"
}'
make_request "POST" "/sessions/$SESSION_ID/send/edit" "$edit_data" "Editar mensagem"

# Testar deletar mensagem
delete_data='{
  "to": "'"$PHONE_NUMBER"'",
  "message_id": "test_text_1756606281",
  "for_everyone": false
}'
make_request "POST" "/sessions/$SESSION_ID/delete" "$delete_data" "Deletar mensagem"

success "🎉 Testes corrigidos finalizados!"
warning "📝 Verifique o número $PHONE_NUMBER para confirmar recebimento"
log "🔧 Session ID: $SESSION_ID"
