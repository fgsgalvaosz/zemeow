package routes

// Este pacote centraliza toda a configuração de rotas da API ZeMeow
// 
// Estrutura de rotas:
// - /health - Health check
// - /sessions - Gerenciamento de sessões WhatsApp
// - /webhooks - Gerenciamento de webhooks
//
// Middleware aplicado:
// - CORS com suporte aos headers de API key
// - Request logging com contexto de sessão
// - Validação automática de requests/responses
// - Autenticação baseada em API keys (global e por sessão)
//
// Autenticação:
// - Global API Key: Acesso total ao sistema
// - Session API Key: Acesso restrito à sessão específica
//
// Headers suportados para API keys:
// - apikey: Método preferido
// - X-API-Key: Método alternativo
// - Authorization: Bearer token (compatibilidade)