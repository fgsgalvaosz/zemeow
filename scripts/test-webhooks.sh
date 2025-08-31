#!/bin/bash

# 🔗 Script para testar webhooks do ZeMeow com Webhook Tester
# Este script automatiza o processo de teste de webhooks

set -e

# Configurações
ZEMEOW_URL="http://localhost:8080"
WEBHOOK_TESTER_URL="http://localhost:8090"
API_KEY="${ADMIN_API_KEY:-test123}"
SESSION_ID="${SESSION_ID:-bd61793a-e353-46b8-8b77-05306a1aa913}"

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔗 ZeMeow Webhook Tester${NC}"
echo "=================================="

# Função para verificar se os serviços estão rodando
check_services() {
    echo -e "${YELLOW}📡 Verificando serviços...${NC}"
    
    # Verificar ZeMeow
    if curl -s "$ZEMEOW_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ ZeMeow está rodando${NC}"
    else
        echo -e "${RED}❌ ZeMeow não está acessível em $ZEMEOW_URL${NC}"
        exit 1
    fi
    
    # Verificar Webhook Tester
    if curl -s "$WEBHOOK_TESTER_URL" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Webhook Tester está rodando${NC}"
    else
        echo -e "${RED}❌ Webhook Tester não está acessível em $WEBHOOK_TESTER_URL${NC}"
        echo -e "${YELLOW}💡 Execute: docker-compose --profile webhook-test up webhook-tester -d${NC}"
        exit 1
    fi
}

# Função para obter uma sessão de webhook
get_webhook_session() {
    echo -e "${YELLOW}🎯 Obtendo sessão de webhook...${NC}"
    
    # Para este exemplo, vamos usar uma sessão fixa
    # Em um ambiente real, você criaria uma nova sessão via interface web
    WEBHOOK_SESSION_ID="zemeow-test-$(date +%s)"
    WEBHOOK_URL="$WEBHOOK_TESTER_URL/webhook/$WEBHOOK_SESSION_ID"
    
    echo -e "${GREEN}📋 URL do Webhook: $WEBHOOK_URL${NC}"
    echo -e "${BLUE}💡 Acesse $WEBHOOK_TESTER_URL para ver as requisições em tempo real${NC}"
}

# Função para configurar webhook no ZeMeow
configure_webhook() {
    echo -e "${YELLOW}⚙️ Configurando webhook no ZeMeow...${NC}"
    
    RESPONSE=$(curl -s -X POST "$ZEMEOW_URL/webhooks/sessions/$SESSION_ID/set" \
        -H "apikey: $API_KEY" \
        -H "Content-Type: application/json" \
        -d "{
            \"url\": \"$WEBHOOK_URL\",
            \"events\": [\"message\", \"receipt\", \"connected\", \"disconnected\"],
            \"active\": true
        }")
    
    if echo "$RESPONSE" | grep -q "configured successfully"; then
        echo -e "${GREEN}✅ Webhook configurado com sucesso${NC}"
    else
        echo -e "${RED}❌ Erro ao configurar webhook:${NC}"
        echo "$RESPONSE"
        exit 1
    fi
}

# Função para testar webhook diretamente
test_webhook_direct() {
    echo -e "${YELLOW}🧪 Testando webhook diretamente...${NC}"
    
    curl -X POST "$WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "{
            \"session_id\": \"$SESSION_ID\",
            \"event\": \"message\",
            \"data\": {
                \"id\": \"msg_test_$(date +%s)\",
                \"from\": \"5511999999999@s.whatsapp.net\",
                \"to\": \"5511888888888@s.whatsapp.net\",
                \"message\": {
                    \"type\": \"text\",
                    \"text\": \"🎉 Hello from ZeMeow! This is a test message sent at $(date)\"
                },
                \"timestamp\": $(date +%s)
            }
        }" > /dev/null 2>&1
    
    echo -e "${GREEN}✅ Webhook de teste enviado${NC}"
}

# Função para listar eventos disponíveis
list_events() {
    echo -e "${YELLOW}📋 Eventos disponíveis no ZeMeow:${NC}"
    
    EVENTS=$(curl -s "$ZEMEOW_URL/webhooks/events" -H "apikey: $API_KEY")
    echo "$EVENTS" | jq -r '.events[] | "• \(.name) (\(.category)) - \(.description)"' 2>/dev/null || echo "$EVENTS"
}

# Função para mostrar estatísticas
show_stats() {
    echo -e "${YELLOW}📊 Buscando webhooks configurados...${NC}"
    
    WEBHOOKS=$(curl -s "$ZEMEOW_URL/webhooks/sessions/$SESSION_ID/find" -H "apikey: $API_KEY")
    echo "$WEBHOOKS" | jq '.' 2>/dev/null || echo "$WEBHOOKS"
}

# Menu principal
show_menu() {
    echo ""
    echo -e "${BLUE}Escolha uma opção:${NC}"
    echo "1. Verificar serviços"
    echo "2. Configurar webhook"
    echo "3. Testar webhook diretamente"
    echo "4. Listar eventos disponíveis"
    echo "5. Mostrar webhooks configurados"
    echo "6. Executar teste completo"
    echo "7. Abrir Webhook Tester no navegador"
    echo "0. Sair"
    echo ""
}

# Função para teste completo
full_test() {
    echo -e "${BLUE}🚀 Executando teste completo...${NC}"
    check_services
    get_webhook_session
    configure_webhook
    test_webhook_direct
    echo ""
    echo -e "${GREEN}🎉 Teste completo finalizado!${NC}"
    echo -e "${BLUE}💡 Acesse $WEBHOOK_TESTER_URL para ver as requisições recebidas${NC}"
}

# Função para abrir no navegador (se disponível)
open_browser() {
    echo -e "${YELLOW}🌐 Tentando abrir Webhook Tester no navegador...${NC}"
    
    if command -v xdg-open > /dev/null; then
        xdg-open "$WEBHOOK_TESTER_URL"
    elif command -v open > /dev/null; then
        open "$WEBHOOK_TESTER_URL"
    else
        echo -e "${BLUE}💡 Abra manualmente: $WEBHOOK_TESTER_URL${NC}"
    fi
}

# Inicialização
get_webhook_session

# Loop do menu
while true; do
    show_menu
    read -p "Digite sua opção: " choice
    
    case $choice in
        1) check_services ;;
        2) configure_webhook ;;
        3) test_webhook_direct ;;
        4) list_events ;;
        5) show_stats ;;
        6) full_test ;;
        7) open_browser ;;
        0) echo -e "${GREEN}👋 Até logo!${NC}"; exit 0 ;;
        *) echo -e "${RED}❌ Opção inválida${NC}" ;;
    esac
    
    echo ""
    read -p "Pressione Enter para continuar..."
done
