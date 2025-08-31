#!/bin/bash

# Script completo para testar TODAS as rotas de chat e grupo da API ZeMeow
BASE_URL="http://localhost:8080"
API_KEY="test123"
PHONE_NUMBER="559984059035"
SESSION_ID="bd61793a-e353-46b8-8b77-05306a1aa913"

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
        echo "  Response: $body" | head -c 200
        echo ""
    else
        error "Status: $status_code"
        echo "  Error: $body"
    fi
    echo ""
}

echo -e "${BLUE}🔥 TESTE COMPLETO DE TODAS AS ROTAS DE CHAT E GRUPO${NC}"
echo "📱 Número: $PHONE_NUMBER"
echo "🔑 Sessão: $SESSION_ID"
echo ""

# Primeiro, vamos enviar uma mensagem para ter um message_id para testar
log "=== PREPARAÇÃO - ENVIANDO MENSAGEM PARA TESTES ==="
text_response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d '{"to": "'$PHONE_NUMBER'", "text": "🧪 Mensagem para testes de chat - ID: test_'$(date +%s)'", "message_id": "test_chat_'$(date +%s)'"}' \
    "$BASE_URL/sessions/$SESSION_ID/send/text")

MESSAGE_ID=$(echo "$text_response" | grep -o '"message_id":"[^"]*"' | cut -d'"' -f4)
if [ -z "$MESSAGE_ID" ]; then
    MESSAGE_ID="test_chat_$(date +%s)"
fi
success "Mensagem enviada com ID: $MESSAGE_ID"
echo ""

# ========================================
# TESTES COMPLETOS DE CHAT
# ========================================

log "=== TESTES COMPLETOS DE FUNCIONALIDADES DE CHAT ==="

# 1. Presença no chat (diferentes tipos)
log "--- TESTES DE PRESENÇA ---"

make_request "POST" "/sessions/$SESSION_ID/chat/presence" \
    '{"to": "'$PHONE_NUMBER'", "presence": "composing"}' \
    "Presença: Digitando (composing)"

sleep 2

make_request "POST" "/sessions/$SESSION_ID/chat/presence" \
    '{"to": "'$PHONE_NUMBER'", "presence": "recording"}' \
    "Presença: Gravando áudio (recording)"

sleep 2

make_request "POST" "/sessions/$SESSION_ID/chat/presence" \
    '{"to": "'$PHONE_NUMBER'", "presence": "paused"}' \
    "Presença: Pausado (paused)"

# 2. Reagir a mensagem
log "--- TESTES DE REAÇÕES ---"

make_request "POST" "/sessions/$SESSION_ID/react" \
    '{"to": "'$PHONE_NUMBER'", "message_id": "'$MESSAGE_ID'", "emoji": "👍"}' \
    "Reagir com 👍"

make_request "POST" "/sessions/$SESSION_ID/react" \
    '{"to": "'$PHONE_NUMBER'", "message_id": "'$MESSAGE_ID'", "emoji": "❤️"}' \
    "Reagir com ❤️"

make_request "POST" "/sessions/$SESSION_ID/react" \
    '{"to": "'$PHONE_NUMBER'", "message_id": "'$MESSAGE_ID'", "emoji": "😂"}' \
    "Reagir com 😂"

# Remover reação
make_request "POST" "/sessions/$SESSION_ID/react" \
    '{"to": "'$PHONE_NUMBER'", "message_id": "'$MESSAGE_ID'", "emoji": ""}' \
    "Remover reação"

# 3. Marcar como lido
log "--- TESTES DE MARCAR COMO LIDO ---"

make_request "POST" "/sessions/$SESSION_ID/chat/markread" \
    '{"to": "'$PHONE_NUMBER'", "message_id": ["'$MESSAGE_ID'"]}' \
    "Marcar mensagem como lida"

# 4. Editar mensagem
log "--- TESTES DE EDIÇÃO DE MENSAGEM ---"

make_request "POST" "/sessions/$SESSION_ID/send/edit" \
    '{"to": "'$PHONE_NUMBER'", "message_id": "'$MESSAGE_ID'", "text": "🔄 Mensagem editada via API - Teste completo!"}' \
    "Editar mensagem"

# 5. Deletar mensagem
log "--- TESTES DE EXCLUSÃO DE MENSAGEM ---"

make_request "POST" "/sessions/$SESSION_ID/delete" \
    '{"to": "'$PHONE_NUMBER'", "message_id": "'$MESSAGE_ID'", "for_everyone": false}' \
    "Deletar mensagem (só para mim)"

# Enviar nova mensagem para deletar para todos
new_text_response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d '{"to": "'$PHONE_NUMBER'", "text": "🗑️ Mensagem para deletar para todos", "message_id": "test_delete_'$(date +%s)'"}' \
    "$BASE_URL/sessions/$SESSION_ID/send/text")

NEW_MESSAGE_ID=$(echo "$new_text_response" | grep -o '"message_id":"[^"]*"' | cut -d'"' -f4)
if [ -n "$NEW_MESSAGE_ID" ]; then
    sleep 2
    make_request "POST" "/sessions/$SESSION_ID/delete" \
        '{"to": "'$PHONE_NUMBER'", "message_id": "'$NEW_MESSAGE_ID'", "for_everyone": true}' \
        "Deletar mensagem (para todos)"
fi

# 6. Download de mídia
log "--- TESTES DE DOWNLOAD DE MÍDIA ---"

make_request "POST" "/sessions/$SESSION_ID/download/image" \
    '{"message_id": "'$MESSAGE_ID'", "path": "/tmp/test_image.jpg"}' \
    "Download de imagem"

make_request "POST" "/sessions/$SESSION_ID/download/video" \
    '{"message_id": "'$MESSAGE_ID'", "path": "/tmp/test_video.mp4"}' \
    "Download de vídeo"

# ========================================
# TESTES COMPLETOS DE GRUPOS
# ========================================

log "=== TESTES COMPLETOS DE FUNCIONALIDADES DE GRUPOS ==="

# 1. Criar grupo para testes
log "--- CRIAÇÃO DE GRUPO PARA TESTES ---"

group_response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d '{"name": "🧪 Grupo Teste Completo API", "participants": ["'$PHONE_NUMBER'"], "description": "Grupo para testes completos da API"}' \
    "$BASE_URL/sessions/$SESSION_ID/group/create")

GROUP_ID=$(echo "$group_response" | grep -o '"group_id":"[^"]*"' | cut -d'"' -f4)
if [ -n "$GROUP_ID" ]; then
    success "Grupo criado com ID: $GROUP_ID"
else
    warning "Falha ao criar grupo, usando ID de exemplo"
    GROUP_ID="120363422342312364@g.us"
fi
echo ""

# 2. Informações do grupo
log "--- TESTES DE INFORMAÇÕES DE GRUPO ---"

make_request "POST" "/sessions/$SESSION_ID/group/info" \
    '{"group_id": "'$GROUP_ID'"}' \
    "Obter informações do grupo"

# 3. Link de convite
log "--- TESTES DE LINK DE CONVITE ---"

make_request "POST" "/sessions/$SESSION_ID/group/invitelink" \
    '{"group_id": "'$GROUP_ID'"}' \
    "Obter link de convite"

# 4. Configurações do grupo
log "--- TESTES DE CONFIGURAÇÕES DE GRUPO ---"

make_request "POST" "/sessions/$SESSION_ID/group/name" \
    '{"group_id": "'$GROUP_ID'", "name": "🔄 Grupo Teste Renomeado"}' \
    "Alterar nome do grupo"

make_request "POST" "/sessions/$SESSION_ID/group/topic" \
    '{"group_id": "'$GROUP_ID'", "topic": "📝 Descrição atualizada via API - Teste completo de funcionalidades"}' \
    "Alterar descrição do grupo"

make_request "POST" "/sessions/$SESSION_ID/group/announce" \
    '{"group_id": "'$GROUP_ID'", "announce": true}' \
    "Ativar modo anúncio (só admins podem enviar)"

make_request "POST" "/sessions/$SESSION_ID/group/announce" \
    '{"group_id": "'$GROUP_ID'", "announce": false}' \
    "Desativar modo anúncio"

make_request "POST" "/sessions/$SESSION_ID/group/locked" \
    '{"group_id": "'$GROUP_ID'", "locked": true}' \
    "Bloquear configurações do grupo"

make_request "POST" "/sessions/$SESSION_ID/group/locked" \
    '{"group_id": "'$GROUP_ID'", "locked": false}' \
    "Desbloquear configurações do grupo"

# 5. Mensagens temporárias
log "--- TESTES DE MENSAGENS TEMPORÁRIAS ---"

make_request "POST" "/sessions/$SESSION_ID/group/ephemeral" \
    '{"group_id": "'$GROUP_ID'", "duration": 86400}' \
    "Ativar mensagens temporárias (24h)"

make_request "POST" "/sessions/$SESSION_ID/group/ephemeral" \
    '{"group_id": "'$GROUP_ID'", "duration": 0}' \
    "Desativar mensagens temporárias"

# 6. Foto do grupo
log "--- TESTES DE FOTO DE GRUPO ---"

make_request "POST" "/sessions/$SESSION_ID/group/photo" \
    '{"group_id": "'$GROUP_ID'", "image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="}' \
    "Definir foto do grupo"

make_request "POST" "/sessions/$SESSION_ID/group/photo/remove" \
    '{"group_id": "'$GROUP_ID'"}' \
    "Remover foto do grupo"

# 7. Informações de convite
log "--- TESTES DE INFORMAÇÕES DE CONVITE ---"

make_request "POST" "/sessions/$SESSION_ID/group/inviteinfo" \
    '{"invite_code": "example_invite_code"}' \
    "Obter informações de convite"

# 8. Sair do grupo
log "--- TESTE DE SAIR DO GRUPO ---"

make_request "POST" "/sessions/$SESSION_ID/group/leave" \
    '{"group_id": "'$GROUP_ID'"}' \
    "Sair do grupo"

# 9. Entrar em grupo via convite
log "--- TESTE DE ENTRAR EM GRUPO ---"

make_request "POST" "/sessions/$SESSION_ID/group/join" \
    '{"invite_code": "example_invite_code"}' \
    "Entrar em grupo via convite"

# ========================================
# TESTES DE MENSAGENS E BULK
# ========================================

log "=== TESTES DE MENSAGENS E BULK ==="

# 1. Listar mensagens
make_request "GET" "/sessions/$SESSION_ID/messages?limit=10&offset=0" "" \
    "Listar mensagens"

# 2. Envio em massa
log "--- TESTE DE ENVIO EM MASSA ---"

make_request "POST" "/sessions/$SESSION_ID/messages/bulk" \
    '{"messages": [{"to": "'$PHONE_NUMBER'", "text": "📨 Mensagem em massa 1"}, {"to": "'$PHONE_NUMBER'", "text": "📨 Mensagem em massa 2"}]}' \
    "Envio de mensagens em massa"

# ========================================
# TESTES DE INFORMAÇÕES E CONTATOS
# ========================================

log "=== TESTES DE INFORMAÇÕES E CONTATOS ==="

make_request "POST" "/sessions/$SESSION_ID/info" \
    '{"phone": "'$PHONE_NUMBER'"}' \
    "Obter informações de contato"

make_request "POST" "/sessions/$SESSION_ID/check" \
    '{"phone": ["'$PHONE_NUMBER'"]}' \
    "Verificar se está no WhatsApp"

make_request "GET" "/sessions/$SESSION_ID/contacts" "" \
    "Listar todos os contatos"

make_request "POST" "/sessions/$SESSION_ID/avatar" \
    '{"phone": "'$PHONE_NUMBER'"}' \
    "Obter avatar do contato"

# ========================================
# TESTES DE NEWSLETTERS
# ========================================

log "=== TESTES DE NEWSLETTERS ==="

make_request "GET" "/sessions/$SESSION_ID/newsletter/list" "" \
    "Listar newsletters"

# ========================================
# RESULTADO FINAL
# ========================================

log "=== TESTE COMPLETO FINALIZADO ==="
success "🎉 Todos os endpoints de chat e grupo foram testados!"
warning "📝 Verifique o número $PHONE_NUMBER e o grupo criado para confirmar as operações"
log "🔧 Session ID: $SESSION_ID"
log "👥 Group ID: $GROUP_ID"
