#!/bin/bash

# Script para configurar latência artificial no sistema
# Uso: ./set-latency.sh [delay_ms]
# Exemplo: ./set-latency.sh 2000  (para 2 segundos de delay)
# Exemplo: ./set-latency.sh 0     (para remover delay)

DELAY_MS=${1:-0}

if ! [[ "$DELAY_MS" =~ ^[0-9]+$ ]]; then
    echo "❌ Erro: Por favor forneça um número válido de milissegundos"
    echo "Uso: $0 [delay_ms]"
    echo "Exemplo: $0 2000  (para 2 segundos de delay)"
    echo "Exemplo: $0 0     (para remover delay)"
    exit 1
fi

echo "🔧 Configurando latência artificial para ${DELAY_MS}ms..."

# Atualizar variável de ambiente no finch-compose.yml
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' "s/PROCESSING_DELAY_MS=[0-9]*/PROCESSING_DELAY_MS=${DELAY_MS}/g" finch-compose.yml
else
    # Linux
    sed -i "s/PROCESSING_DELAY_MS=[0-9]*/PROCESSING_DELAY_MS=${DELAY_MS}/g" finch-compose.yml
fi

echo "📝 Arquivo finch-compose.yml atualizado"

# Reiniciar serviços para aplicar a nova configuração
echo "🔄 Reiniciando serviços para aplicar nova configuração..."
finch compose -f finch-compose.yml down > /dev/null 2>&1
finch compose -f finch-compose.yml up -d > /dev/null 2>&1

echo "⏳ Aguardando serviços iniciarem..."
sleep 15

# Verificar se os serviços estão rodando
echo "🔍 Verificando status dos serviços..."
healthy_count=0
for port in 4333 8080 8084 4444 8087; do
    if curl -s http://localhost:$port/health > /dev/null 2>&1; then
        healthy_count=$((healthy_count + 1))
    fi
done

if [ $healthy_count -eq 5 ]; then
    echo "✅ Todos os serviços estão saudáveis!"
    if [ $DELAY_MS -eq 0 ]; then
        echo "🚀 Latência artificial removida - sistema rodando em velocidade normal"
    else
        echo "⏱️  Latência artificial configurada para ${DELAY_MS}ms"
        echo "💡 Agora você pode executar ./test-complete-flow.sh e monitorar com ./dashboard.sh"
        echo "💡 Para ver dados sendo processados em tempo real, execute em terminais separados:"
        echo "   Terminal 1: ./dashboard.sh"
        echo "   Terminal 2: ./test-complete-flow.sh"
    fi
else
    echo "⚠️  Alguns serviços podem não estar totalmente prontos ainda"
    echo "💡 Execute 'make health' para verificar o status detalhado"
fi

echo ""
echo "📊 Para monitorar o sistema:"
echo "   ./dashboard.sh          - Dashboard em tempo real"
echo "   ./test-complete-flow.sh - Executar teste completo"
echo ""
echo "🔧 Para alterar a latência novamente:"
echo "   $0 [novo_delay_ms]"
