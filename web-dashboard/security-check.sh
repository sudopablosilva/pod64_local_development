#!/bin/bash

# Script de verificação de segurança para o Web Dashboard
# Uso: ./security-check.sh

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_check() {
    echo -e "${BLUE}[CHECK]${NC} $1"
}

echo "🔒 Verificação de Segurança - Web Dashboard"
echo "=========================================="

# Verificar se estamos no diretório correto
if [ ! -f "package.json" ]; then
    log_error "Execute este script no diretório web-dashboard"
    exit 1
fi

# 1. Verificar versão do Node.js
log_check "Verificando versão do Node.js..."
NODE_VERSION=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -ge 22 ]; then
    log_info "✅ Node.js v$NODE_VERSION (recomendado: ≥22)"
else
    log_warn "⚠️  Node.js v$NODE_VERSION (recomendado: ≥22)"
fi

# 2. Auditoria de segurança do backend
log_check "Executando auditoria de segurança do backend..."
if npm audit --audit-level=high > /dev/null 2>&1; then
    log_info "✅ Backend: Nenhuma vulnerabilidade crítica encontrada"
else
    log_warn "⚠️  Backend: Vulnerabilidades encontradas - execute 'npm audit' para detalhes"
fi

# 3. Auditoria de segurança do frontend
log_check "Executando auditoria de segurança do frontend..."
cd frontend
if npm audit --audit-level=high > /dev/null 2>&1; then
    log_info "✅ Frontend: Nenhuma vulnerabilidade crítica encontrada"
else
    log_warn "⚠️  Frontend: Vulnerabilidades encontradas - execute 'npm audit' para detalhes"
fi
cd ..

# 4. Verificar configurações de segurança
log_check "Verificando configurações de segurança..."

# Verificar .npmrc
if [ -f ".npmrc" ]; then
    if grep -q "audit-level=high" .npmrc; then
        log_info "✅ Configuração .npmrc: audit-level=high configurado"
    else
        log_warn "⚠️  Configuração .npmrc: audit-level não configurado"
    fi
    
    if grep -q "strict-ssl=true" .npmrc; then
        log_info "✅ Configuração .npmrc: strict-ssl habilitado"
    else
        log_warn "⚠️  Configuração .npmrc: strict-ssl não configurado"
    fi
else
    log_warn "⚠️  Arquivo .npmrc não encontrado"
fi

# 5. Verificar Dockerfile
log_check "Verificando Dockerfile..."
if grep -q "node:22-alpine" Dockerfile; then
    log_info "✅ Dockerfile: Usando Node.js 22 (versão segura)"
else
    log_warn "⚠️  Dockerfile: Não está usando Node.js 22"
fi

if grep -q "npm audit fix" Dockerfile; then
    log_info "✅ Dockerfile: Auditoria automática configurada"
else
    log_warn "⚠️  Dockerfile: Auditoria automática não configurada"
fi

# 6. Verificar dependências desatualizadas
log_check "Verificando dependências desatualizadas..."
OUTDATED_BACKEND=$(npm outdated --json 2>/dev/null | jq -r 'keys[]' 2>/dev/null | wc -l)
cd frontend
OUTDATED_FRONTEND=$(npm outdated --json 2>/dev/null | jq -r 'keys[]' 2>/dev/null | wc -l)
cd ..

if [ "$OUTDATED_BACKEND" -eq 0 ]; then
    log_info "✅ Backend: Todas as dependências estão atualizadas"
else
    log_warn "⚠️  Backend: $OUTDATED_BACKEND dependências desatualizadas"
fi

if [ "$OUTDATED_FRONTEND" -eq 0 ]; then
    log_info "✅ Frontend: Todas as dependências estão atualizadas"
else
    log_warn "⚠️  Frontend: $OUTDATED_FRONTEND dependências desatualizadas"
fi

# 7. Verificar se o serviço está rodando com segurança
log_check "Verificando configuração do serviço..."
if curl -s http://localhost:3000/health > /dev/null 2>&1; then
    log_info "✅ Serviço: Respondendo corretamente"
    
    # Verificar headers de segurança
    SECURITY_HEADERS=$(curl -s -I http://localhost:3000 | grep -E "(X-Frame-Options|X-Content-Type-Options|X-XSS-Protection|Strict-Transport-Security)" | wc -l)
    if [ "$SECURITY_HEADERS" -ge 2 ]; then
        log_info "✅ Headers de segurança: Configurados ($SECURITY_HEADERS headers encontrados)"
    else
        log_warn "⚠️  Headers de segurança: Poucos headers configurados ($SECURITY_HEADERS/4)"
    fi
else
    log_warn "⚠️  Serviço: Não está respondendo (pode estar offline)"
fi

# 8. Resumo final
echo ""
echo "📊 Resumo da Verificação de Segurança"
echo "====================================="

# Contar warnings e erros
WARNINGS=$(grep -c "⚠️" /tmp/security_check_output 2>/dev/null || echo "0")
ERRORS=$(grep -c "❌" /tmp/security_check_output 2>/dev/null || echo "0")

if [ "$WARNINGS" -eq 0 ] && [ "$ERRORS" -eq 0 ]; then
    log_info "🎉 Excelente! Nenhum problema de segurança encontrado."
elif [ "$ERRORS" -eq 0 ]; then
    log_warn "⚠️  $WARNINGS avisos encontrados. Considere as recomendações acima."
else
    log_error "❌ $ERRORS erros e $WARNINGS avisos encontrados. Ação necessária!"
fi

echo ""
echo "🔧 Comandos úteis:"
echo "   ./update-dependencies.sh    # Atualizar dependências"
echo "   npm audit                   # Ver detalhes de vulnerabilidades"
echo "   npm outdated               # Ver dependências desatualizadas"
echo "   finch compose build web-dashboard  # Rebuild do container"
