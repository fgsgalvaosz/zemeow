#!/bin/bash

# Script para corrigir os handlers que não estão usando a função helper getWhatsAppClient

echo "🔧 Corrigindo handlers para usar getWhatsAppClient helper..."

# Função para fazer backup
backup_file() {
    local file=$1
    cp "$file" "$file.backup.$(date +%s)"
    echo "✅ Backup criado: $file.backup.$(date +%s)"
}

# Corrigir MessageHandler - SendText (que funciona)
echo "📝 Verificando SendText que funciona..."
grep -n "getWhatsAppClient\|GetWhatsAppClient" internal/api/handlers/message.go | head -5

echo ""
echo "📝 Verificando outros métodos que não funcionam..."
grep -n -A 10 "SendMedia\|SendLocation\|SendContact" internal/api/handlers/message.go | grep -A 5 "GetWhatsAppClient"

echo ""
echo "🔍 Analisando diferenças entre SendText (funciona) e SendMedia (não funciona)..."

# Verificar se SendText usa getWhatsAppClient helper
echo "SendText usa helper?"
grep -A 20 "func (h \*MessageHandler) SendText" internal/api/handlers/message.go | grep -E "(getWhatsAppClient|GetWhatsAppClient)"

echo ""
echo "SendMedia usa helper?"
grep -A 20 "func (h \*MessageHandler) SendMedia" internal/api/handlers/message.go | grep -E "(getWhatsAppClient|GetWhatsAppClient)"

echo ""
echo "🔧 Vamos verificar se o problema está na implementação dos handlers..."

# Verificar se todos os métodos estão usando o mesmo padrão
echo "Métodos que usam GetWhatsAppClient diretamente:"
grep -n "GetWhatsAppClient.*sessionID" internal/api/handlers/message.go

echo ""
echo "Métodos que usam getWhatsAppClient helper:"
grep -n "getWhatsAppClient.*sessionID" internal/api/handlers/message.go

echo ""
echo "🔍 Verificando implementação do helper getWhatsAppClient..."
grep -A 30 "func (h \*MessageHandler) getWhatsAppClient" internal/api/handlers/message.go

echo ""
echo "🔍 Verificando se há diferenças na implementação entre handlers..."

# Comparar GroupHandler
echo "GroupHandler getWhatsAppClient:"
grep -A 20 "func (h \*GroupHandler) getWhatsAppClient" internal/api/handlers/group.go

echo ""
echo "📊 Resumo dos problemas encontrados:"
echo "1. Alguns handlers usam GetWhatsAppClient diretamente"
echo "2. Outros usam getWhatsAppClient helper"
echo "3. O cast direto para *whatsmeow.Client falha"
echo "4. Precisa usar o método GetClient() do MyClient"

echo ""
echo "🎯 Solução: Todos os handlers devem usar getWhatsAppClient helper"
echo "   que implementa o fallback para GetClient() quando o cast direto falha"

echo ""
echo "📝 Verificando quais métodos precisam ser corrigidos..."

# Lista de métodos que provavelmente precisam correção
methods=(
    "SendMedia"
    "SendLocation" 
    "SendContact"
    "SendSticker"
    "SendButtons"
    "SendList"
    "SendPoll"
    "EditMessage"
    "DeleteMessage"
    "ReactToMessage"
    "SetChatPresence"
    "MarkAsRead"
)

for method in "${methods[@]}"; do
    echo "Verificando $method..."
    if grep -q "func (h \*MessageHandler) $method" internal/api/handlers/message.go; then
        uses_helper=$(grep -A 20 "func (h \*MessageHandler) $method" internal/api/handlers/message.go | grep -c "getWhatsAppClient")
        uses_direct=$(grep -A 20 "func (h \*MessageHandler) $method" internal/api/handlers/message.go | grep -c "GetWhatsAppClient")
        
        if [ $uses_helper -eq 0 ] && [ $uses_direct -gt 0 ]; then
            echo "  ❌ $method usa GetWhatsAppClient direto - PRECISA CORREÇÃO"
        elif [ $uses_helper -gt 0 ]; then
            echo "  ✅ $method usa getWhatsAppClient helper - OK"
        else
            echo "  ❓ $method - padrão não identificado"
        fi
    else
        echo "  ⚠️  $method não encontrado"
    fi
done

echo ""
echo "🔧 Para corrigir, substitua em todos os handlers:"
echo "   clientInterface, err := h.sessionService.GetWhatsAppClient(context.Background(), sessionID)"
echo "   client, ok := clientInterface.(*whatsmeow.Client)"
echo ""
echo "Por:"
echo "   client, err := h.getWhatsAppClient(sessionID)"
echo ""

echo "✅ Análise completa! Execute os testes novamente após as correções."
