#!/bin/bash

# Script para atualizar imports após reorganização da estrutura

echo "Atualizando imports nos arquivos..."

# Atualizar imports de db/models para models
find internal/ -name "*.go" -type f -exec sed -i 's|github.com/felipe/zemeow/internal/db/models|github.com/felipe/zemeow/internal/models|g' {} \;

# Atualizar imports de db/repositories para repositories  
find internal/ -name "*.go" -type f -exec sed -i 's|github.com/felipe/zemeow/internal/db/repositories|github.com/felipe/zemeow/internal/repositories|g' {} \;

# Atualizar imports de service/ para services/
find internal/ -name "*.go" -type f -exec sed -i 's|github.com/felipe/zemeow/internal/service/|github.com/felipe/zemeow/internal/services/|g' {} \;

# Atualizar imports de api/dto para dto
find internal/ -name "*.go" -type f -exec sed -i 's|github.com/felipe/zemeow/internal/api/dto|github.com/felipe/zemeow/internal/dto|g' {} \;

# Atualizar imports de api/middleware para middleware
find internal/ -name "*.go" -type f -exec sed -i 's|github.com/felipe/zemeow/internal/api/middleware|github.com/felipe/zemeow/internal/middleware|g' {} \;

# Atualizar imports de api/validators para middleware  
find internal/ -name "*.go" -type f -exec sed -i 's|github.com/felipe/zemeow/internal/api/validators|github.com/felipe/zemeow/internal/middleware|g' {} \;

# Atualizar imports de api/utils para utils (manter no handlers por enquanto)
find internal/ -name "*.go" -type f -exec sed -i 's|github.com/felipe/zemeow/internal/api/utils|github.com/felipe/zemeow/internal/handlers/utils|g' {} \;

# Atualizar imports de api/handlers para handlers
find internal/ -name "*.go" -type f -exec sed -i 's|github.com/felipe/zemeow/internal/api/handlers|github.com/felipe/zemeow/internal/handlers|g' {} \;

# Atualizar imports de db para database
find internal/ -name "*.go" -type f -exec sed -i 's|github.com/felipe/zemeow/internal/db|github.com/felipe/zemeow/internal/database|g' {} \;

echo "Imports atualizados com sucesso!"