#!/bin/bash

# Dashboard em tempo real para monitorar o sistema POC BDD
# Mostra contadores de tabelas DynamoDB e filas SQS com dados versionados
# ATUALIZADO: Usa endpoints dos microserviços em vez de acesso direto ao LocalStack

echo "🚀 POC BDD - Dashboard em Tempo Real (Versão Corrigida)"
echo "Pressione Ctrl+C para sair"
echo ""

while true; do
    clear
    echo "📊 POC BDD - Dashboard em Tempo Real - $(date '+%H:%M:%S')"
    echo "=============================================================="
    echo ""
    
    echo "📋 TABELAS DYNAMODB (via microserviços):"
    echo "----------------------------------------------"
    
    # Tabela executions via JMI (usando endpoint corrigido)
    executions_data=$(curl -s http://localhost:4333/executions 2>/dev/null)
    executions_count=$(echo "$executions_data" | jq -r '.count // 0' 2>/dev/null)
    
    if [ "$executions_count" = "null" ] || [ -z "$executions_count" ]; then
        executions_count="❌ JMI offline"
    fi
    
    echo "🔄 executions: $executions_count registros (via JMI)"
    
    # Mostrar últimos registros de executions se houver dados
    if [ "$executions_count" != "❌ JMI offline" ] && [ "$executions_count" -gt 0 ] 2>/dev/null; then
        echo "   📝 Últimos registros versionados:"
        echo "$executions_data" | jq -r '.executions[]? | "      • " + (.executionName // "N/A") + " | " + (.stage // "N/A") + " | " + (.status // "N/A")' 2>/dev/null | head -5
    fi
    
    # Verificar outras tabelas via JMI
    tables_data=$(curl -s http://localhost:4333/tables 2>/dev/null)
    tables_count=$(echo "$tables_data" | jq -r '.count // 0' 2>/dev/null)
    
    if [ "$tables_count" != "0" ] && [ "$tables_count" != "null" ] 2>/dev/null; then
        echo "📋 Total de tabelas disponíveis: $tables_count"
        echo "   Tabelas: $(echo "$tables_data" | jq -r '.tables[]?' 2>/dev/null | tr '\n' ' ')"
    else
        echo "📋 ⚠️ Não foi possível listar outras tabelas"
    fi
    
    echo ""
    echo "📨 FILAS SQS (via JMI):"
    echo "----------------------------------------------"
    
    # Obter informações das filas via JMI
    queues_data=$(curl -s http://localhost:4333/queues 2>/dev/null)
    queues_count=$(echo "$queues_data" | jq -r '.count // 0' 2>/dev/null)
    
    if [ "$queues_count" != "0" ] && [ "$queues_count" != "null" ] 2>/dev/null; then
        echo "📬 Total de filas: $queues_count"
        echo "$queues_data" | jq -r '.queues[]? | "   • " + .name + ": " + (.visibleMessages // "0") + " visíveis, " + (.notVisibleMessages // "0") + " processando"' 2>/dev/null
    else
        echo "📬 ⚠️ Não foi possível acessar informações das filas SQS"
    fi
    echo "----------------------------------------------"
    
    # Verificar status de cada serviço
    services=(
        "Control-M:4333"
        "JMI:4333" 
        "JMW:8080"
        "JMR:8084"
        "Scheduler:8085"
        "SPA:4444"
        "SPAQ:8087"
    )
    
    for service_info in "${services[@]}"; do
        service_name=$(echo "$service_info" | cut -d: -f1)
        service_port=$(echo "$service_info" | cut -d: -f2)
        
        if curl -s http://localhost:$service_port/health > /dev/null 2>&1; then
            echo "✅ $service_name (porta $service_port): Online"
        else
            echo "❌ $service_name (porta $service_port): Offline"
        fi
    done
    
    echo ""
    echo "🏥 STATUS DOS SERVIÇOS:"
    
    echo ""
    echo "🔧 CONFIGURAÇÃO ATUAL:"
    echo "----------------------------------------------"
    
    # Verificar latência configurada
    current_delay=$(grep "PROCESSING_DELAY_MS=" finch-compose.yml | head -1 | sed 's/.*PROCESSING_DELAY_MS=\([0-9]*\).*/\1/')
    if [ "$current_delay" = "0" ] || [ -z "$current_delay" ]; then
        echo "⚡ Latência: Desabilitada (processamento em velocidade normal)"
    else
        echo "⏱️  Latência: ${current_delay}ms por operação"
    fi
    
    echo ""
    echo "💡 COMANDOS ÚTEIS (ATUALIZADOS):"
    echo "----------------------------------------------"
    echo "• ./set-latency.sh 3000           - Configurar 3s de latência"
    echo "• ./set-latency.sh 0              - Remover latência"
    echo "• ./test-complete-flow.sh         - Executar teste completo"
    echo ""
    echo "📊 ENDPOINTS PARA MONITORAMENTO:"
    echo "----------------------------------------------"
    echo "• curl http://localhost:4333/tables      - Listar tabelas DynamoDB"
    echo "• curl http://localhost:4333/executions  - Ver execuções (JMI)"
    echo "• curl http://localhost:4333/queues     - Ver status das filas SQS"
    echo "• curl http://localhost:4333/health      - Status do JMI"
    echo ""
    echo "Atualizando em 5 segundos... (Ctrl+C para sair)"
    
    sleep 5
done
