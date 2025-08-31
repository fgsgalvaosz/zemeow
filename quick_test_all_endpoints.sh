#!/bin/bash

# Script de teste rápido para validar todos os endpoints principais da API ZeMeow
# Uso: ./quick_test_all_endpoints.sh [PHONE_NUMBER] [SESSION_ID]

# Configurações padrão
BASE_URL="http://localhost:8080"
API_KEY="test123"
PHONE_NUMBER="${1:-559984059035}"
SESSION_ID="${2:-bd61793a-e353-46b8-8b77-05306a1aa913}"

# Cores
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}🚀 TESTE RÁPIDO DA API ZEMEOW${NC}"
echo "📱 Número: $PHONE_NUMBER"
echo "🔑 Sessão: $SESSION_ID"
echo "🌐 URL: $BASE_URL"
echo ""

# Função para teste rápido
quick_test() {
    local method=$1
    local endpoint=$2
    local data=$3
    local name=$4
    
    if [ -n "$data" ]; then
        status=$(curl -s -w "%{http_code}" -o /dev/null -X "$method" \
            -H "Content-Type: application/json" \
            -H "X-API-Key: $API_KEY" \
            -d "$data" \
            "$BASE_URL$endpoint")
    else
        status=$(curl -s -w "%{http_code}" -o /dev/null -X "$method" \
            -H "X-API-Key: $API_KEY" \
            "$BASE_URL$endpoint")
    fi
    
    if [[ $status -ge 200 && $status -lt 300 ]]; then
        echo -e "${GREEN}✅ $name${NC} (${status})"
        return 0
    else
        echo -e "${RED}❌ $name${NC} (${status})"
        return 1
    fi
}

# Contador de sucessos
success_count=0
total_count=0

# 1. Health Check
echo -e "${BLUE}=== BÁSICOS ===${NC}"
quick_test "GET" "/health" "" "Health Check" && ((success_count++))
((total_count++))

quick_test "GET" "/sessions" "" "Listar Sessões" && ((success_count++))
((total_count++))

quick_test "GET" "/sessions/$SESSION_ID/status" "" "Status da Sessão" && ((success_count++))
((total_count++))

# 2. Envio de Mensagens
echo -e "${BLUE}=== ENVIO DE MENSAGENS ===${NC}"

quick_test "POST" "/sessions/$SESSION_ID/send/text" \
    '{"to": "'$PHONE_NUMBER'", "text": "🤖 Teste rápido da API"}' \
    "Enviar Texto" && ((success_count++))
((total_count++))

quick_test "POST" "/sessions/$SESSION_ID/send/media" \
    '{"to": "'$PHONE_NUMBER'", "type": "image", "media": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==", "caption": "📸 Teste"}' \
    "Enviar Mídia" && ((success_count++))
((total_count++))

quick_test "POST" "/sessions/$SESSION_ID/send/location" \
    '{"to": "'$PHONE_NUMBER'", "latitude": -23.550520, "longitude": -46.633308, "name": "📍 São Paulo"}' \
    "Enviar Localização" && ((success_count++))
((total_count++))

quick_test "POST" "/sessions/$SESSION_ID/send/contact" \
    '{"to": "'$PHONE_NUMBER'", "name": "👤 Contato Teste", "vcard": "BEGIN:VCARD\nVERSION:3.0\nFN:Contato Teste\nTEL;TYPE=CELL:+5511999999999\nEND:VCARD"}' \
    "Enviar Contato" && ((success_count++))
((total_count++))

quick_test "POST" "/sessions/$SESSION_ID/send/sticker" \
    '{"to": "'$PHONE_NUMBER'", "sticker": "data:image/webp;base64,UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA"}' \
    "Enviar Sticker" && ((success_count++))
((total_count++))

quick_test "POST" "/sessions/$SESSION_ID/send/buttons" \
    '{"to": "'$PHONE_NUMBER'", "text": "🔘 Escolha:", "buttons": [{"id": "btn1", "text": "✅ Sim"}, {"id": "btn2", "text": "❌ Não"}]}' \
    "Enviar Botões" && ((success_count++))
((total_count++))

quick_test "POST" "/sessions/$SESSION_ID/send/list" \
    '{"to": "'$PHONE_NUMBER'", "text": "📋 Lista:", "title": "Opções", "button_text": "Ver", "sections": [{"title": "Seção", "rows": [{"id": "1", "title": "Item 1", "description": "Desc 1"}]}]}' \
    "Enviar Lista" && ((success_count++))
((total_count++))

quick_test "POST" "/sessions/$SESSION_ID/send/poll" \
    '{"to": "'$PHONE_NUMBER'", "name": "🗳️ Cor favorita?", "options": ["🔵 Azul", "🟢 Verde"], "selectable": 1}' \
    "Enviar Enquete" && ((success_count++))
((total_count++))

# 3. Chat
echo -e "${BLUE}=== FUNCIONALIDADES DE CHAT ===${NC}"

quick_test "POST" "/sessions/$SESSION_ID/presence" \
    '{"presence": "available"}' \
    "Definir Presença" && ((success_count++))
((total_count++))

# 4. Grupos
echo -e "${BLUE}=== GRUPOS ===${NC}"

quick_test "POST" "/sessions/$SESSION_ID/group/create" \
    '{"name": "🤖 Teste API", "participants": ["'$PHONE_NUMBER'"], "description": "Grupo de teste"}' \
    "Criar Grupo" && ((success_count++))
((total_count++))

quick_test "GET" "/sessions/$SESSION_ID/group/list" "" "Listar Grupos" && ((success_count++))
((total_count++))

# 5. Informações
echo -e "${BLUE}=== INFORMAÇÕES ===${NC}"

quick_test "POST" "/sessions/$SESSION_ID/check" \
    '{"phone": ["'$PHONE_NUMBER'"]}' \
    "Verificar Contatos" && ((success_count++))
((total_count++))

quick_test "GET" "/sessions/$SESSION_ID/contacts" "" "Listar Contatos" && ((success_count++))
((total_count++))

quick_test "POST" "/sessions/$SESSION_ID/avatar" \
    '{"phone": "'$PHONE_NUMBER'"}' \
    "Obter Avatar" && ((success_count++))
((total_count++))

# 6. Estatísticas
echo -e "${BLUE}=== ESTATÍSTICAS ===${NC}"

quick_test "GET" "/sessions/$SESSION_ID/stats" "" "Estatísticas" && ((success_count++))
((total_count++))

# Resultado final
echo ""
echo -e "${BLUE}=== RESULTADO FINAL ===${NC}"

percentage=$((success_count * 100 / total_count))

if [ $percentage -ge 80 ]; then
    color=$GREEN
    status="🎉 EXCELENTE"
elif [ $percentage -ge 60 ]; then
    color=$YELLOW
    status="⚠️ BOM"
else
    color=$RED
    status="❌ PRECISA MELHORAR"
fi

echo -e "${color}$status${NC}"
echo "✅ Sucessos: $success_count/$total_count ($percentage%)"
echo "📱 Número testado: $PHONE_NUMBER"
echo "🔑 Sessão: $SESSION_ID"

if [ $percentage -ge 80 ]; then
    echo ""
    echo -e "${GREEN}🚀 API ZeMeow está funcionando perfeitamente!${NC}"
    echo -e "${BLUE}📝 Verifique o WhatsApp do número $PHONE_NUMBER para confirmar as mensagens${NC}"
else
    echo ""
    echo -e "${YELLOW}⚠️ Alguns endpoints precisam de atenção${NC}"
    echo -e "${BLUE}📋 Execute o teste detalhado para mais informações:${NC}"
    echo "   ./test_all_endpoints.sh"
fi

echo ""
echo -e "${BLUE}📊 Para relatório completo, veja: FINAL_TEST_REPORT.md${NC}"

exit $((total_count - success_count))
