#!/bin/bash

# Script para atualizar dependências com segurança
# Uso: ./update-dependencies.sh

set -e

echo "🔄 Atualizando dependências do Web Dashboard..."

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Função para log colorido
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Verificar se estamos no diretório correto
if [ ! -f "package.json" ]; then
    log_error "Execute este script no diretório web-dashboard"
    exit 1
fi

# Backup dos package-lock.json
log_info "Fazendo backup dos arquivos de lock..."
cp package-lock.json package-lock.json.backup 2>/dev/null || true
cp frontend/package-lock.json frontend/package-lock.json.backup 2>/dev/null || true

# Atualizar dependências do backend
log_info "Atualizando dependências do backend..."
rm -rf node_modules package-lock.json
npm install
npm audit fix --audit-level=high || log_warn "Algumas vulnerabilidades podem precisar de atenção manual"

# Atualizar dependências do frontend
log_info "Atualizando dependências do frontend..."
cd frontend
rm -rf node_modules package-lock.json
npm install
npm audit fix --audit-level=high || log_warn "Algumas vulnerabilidades podem precisar de atenção manual"
cd ..

# Executar auditoria de segurança
log_info "Executando auditoria de segurança..."
npm run security:audit || log_warn "Vulnerabilidades encontradas - verifique manualmente"
cd frontend && npm run security:audit || log_warn "Vulnerabilidades no frontend - verifique manualmente"
cd ..

# Testar se a aplicação ainda funciona
log_info "Testando build do frontend..."
cd frontend && npm run build && cd ..

log_info "✅ Atualização concluída!"
log_info "📋 Próximos passos:"
echo "   1. Teste a aplicação localmente"
echo "   2. Execute os testes: npm test"
echo "   3. Rebuild o container: finch compose build web-dashboard"
echo "   4. Se houver problemas, restaure os backups:"
echo "      - mv package-lock.json.backup package-lock.json"
echo "      - mv frontend/package-lock.json.backup frontend/package-lock.json"
