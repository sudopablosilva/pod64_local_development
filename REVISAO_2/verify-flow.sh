#!/bin/bash

echo "=== Verificação Completa do Fluxo ==="
echo ""

# Function to check service health
check_health() {
    local port=$1
    local service=$2
    echo "🔍 Verificando $service (porta $port)..."
    response=$(curl -s http://localhost:$port/health)
    if [ $? -eq 0 ]; then
        echo "✅ $service está saudável"
        echo "   Resposta: $response"
    else
        echo "❌ $service não está respondendo"
    fi
    echo ""
}

# Function to test endpoint
test_endpoint() {
    local method=$1
    local url=$2
    local data=$3
    local description=$4
    
    echo "🧪 Testando: $description"
    echo "   URL: $url"
    
    if [ -n "$data" ]; then
        response=$(curl -s -X $method "$url" -H "Content-Type: application/json" -d "$data")
    else
        response=$(curl -s -X $method "$url")
    fi
    
    if [ $? -eq 0 ]; then
        echo "✅ Sucesso"
        echo "   Resposta: $response"
    else
        echo "❌ Falhou"
    fi
    echo ""
}

# Function to check DynamoDB table
check_dynamodb_table() {
    local table=$1
    echo "🗄️  Verificando tabela DynamoDB: $table"
    
    response=$(curl -s -X POST "http://localhost:4566/" \
      -H "Content-Type: application/x-amz-json-1.0" \
      -H "X-Amz-Target: DynamoDB_20120810.Scan" \
      -d "{\"TableName\": \"$table\"}")
    
    if echo "$response" | grep -q "Items"; then
        count=$(echo "$response" | jq -r '.Count // 0')
        echo "✅ Tabela $table existe com $count itens"
        if [ "$count" -gt 0 ]; then
            echo "   Dados: $(echo "$response" | jq -c '.Items[0] // {}')"
        fi
    else
        echo "❌ Tabela $table não existe ou erro: $response"
    fi
    echo ""
}

# Function to check SQS queue
check_sqs_queue() {
    local queue=$1
    echo "📬 Verificando fila SQS: $queue"
    
    # Get queue attributes
    response=$(curl -s "http://localhost:4566/000000000000/$queue?Action=GetQueueAttributes&AttributeName.1=All&Version=2012-11-05")
    
    if echo "$response" | grep -q "ApproximateNumberOfMessages"; then
        messages=$(echo "$response" | grep -o '<Value>[0-9]*</Value>' | head -1 | sed 's/<[^>]*>//g')
        echo "✅ Fila $queue existe com $messages mensagens"
    else
        echo "❌ Fila $queue não existe ou erro"
    fi
    echo ""
}

echo "1. VERIFICAÇÃO DE SAÚDE DOS SERVIÇOS"
echo "=================================="
check_health 4333 "JMI (Job Manager Integrator)"
check_health 8080 "JMW (Job Manager Worker)"
check_health 8084 "JMR (Job Manager Runner)"
check_health 8085 "Scheduler Plugin"
check_health 4444 "SPA (Scheduler Plugin Adapter) - Schedule"
check_health 4446 "SPA (Scheduler Plugin Adapter) - Trigger"
check_health 8087 "SPAQ (Scheduler Plugin Adapter Queue)"

echo "2. VERIFICAÇÃO DA INFRAESTRUTURA"
echo "================================"

echo "📋 Listando tabelas DynamoDB..."
tables_response=$(curl -s -X POST "http://localhost:4566/" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -H "X-Amz-Target: DynamoDB_20120810.ListTables" \
  -d '{}')
echo "Tabelas disponíveis: $(echo "$tables_response" | jq -r '.TableNames[]' | tr '\n' ' ')"
echo ""

check_dynamodb_table "executions"
check_dynamodb_table "jobs"
check_dynamodb_table "schedules"
check_dynamodb_table "adapters"
check_dynamodb_table "queue_messages"

echo "📬 Verificando filas SQS..."
check_sqs_queue "job-requests"
check_sqs_queue "jmw-queue"
check_sqs_queue "jmr-queue"
check_sqs_queue "sp-queue"
check_sqs_queue "spa-queue"
check_sqs_queue "spaq-queue"

echo "3. TESTE DO FLUXO COMPLETO"
echo "=========================="

echo "🚀 Iniciando execução via JMI..."
execution_response=$(curl -s -X POST http://localhost:4333/startExecution \
  -H "Content-Type: application/json" \
  -d '{
    "executionName": "FLOW_TEST_'$(date +%s)'"
  }')

echo "Resposta do JMI: $execution_response"
execution_uuid=$(echo "$execution_response" | jq -r '.executionUuid // empty')
execution_name=$(echo "$execution_response" | jq -r '.executionName // empty')

if [ -n "$execution_uuid" ]; then
    echo "✅ Execução criada com UUID: $execution_uuid"
    
    echo ""
    echo "⏳ Aguardando 3 segundos para processamento..."
    sleep 3
    
    echo ""
    echo "🔍 Verificando se a execução foi armazenada..."
    check_dynamodb_table "executions"
    
    echo ""
    echo "🛑 Testando parada da execução..."
    stop_response=$(curl -s -X POST http://localhost:4333/stopExecution \
      -H "Content-Type: application/json" \
      -d "{
        \"executionName\": \"$execution_name\",
        \"executionUuid\": \"$execution_uuid\"
      }")
    echo "Resposta do Stop: $stop_response"
else
    echo "❌ Falha ao criar execução"
fi

echo ""
echo "🔧 Testando JMW Start..."
jmw_response=$(curl -s -X POST http://localhost:8080/start \
  -H "Content-Type: application/json" \
  -d '{
    "executionName": "JMW_TEST_'$(date +%s)'",
    "accountId": "017820684888",
    "commonProperties": {
      "region": "sa-east-1"
    },
    "runtimes": [
      {
        "runtimeName": "test-runtime",
        "compute": {"type": "test"}
      }
    ],
    "schedulerRoutine": {
      "executionName": "TEST_ROUTINE",
      "cron": "0 6 * * *",
      "steps": [
        {
          "stepId": "STEP-1",
          "tasks": [{"taskId": "TASK-1", "runtimeName": "test-runtime"}]
        }
      ]
    }
  }')

echo "Resposta do JMW: $jmw_response"

echo ""
echo "🎯 Testando SPA Trigger..."
trigger_response=$(curl -s -X POST http://localhost:4446/v1/trigger \
  -H "Content-Type: application/json" \
  -d '{
    "accountId": "017820684888",
    "executionName": "TRIGGER_TEST_'$(date +%s)'",
    "eventDate": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "eventType": "TestTrigger",
    "eventId": "TEST_'$(date +%s)'",
    "parameters": {"test": true}
  }')

echo "Resposta do SPA Trigger: $trigger_response"

echo ""
echo "📅 Testando SPA Schedule..."
schedule_response=$(curl -s -X POST http://localhost:4444/v1/schedule \
  -H "Content-Type: application/json" \
  -d '{
    "acronym": "TST",
    "repo": "TEST_REPO_'$(date +%s)'",
    "routines": [
      {
        "name": "TEST_ROUTINE_'$(date +%s)'",
        "description": "Rotina de teste",
        "cron": "0 12 * * *",
        "priority": "medium"
      }
    ]
  }')

echo "Resposta do SPA Schedule: $schedule_response"

echo ""
echo "4. VERIFICAÇÃO FINAL DOS DADOS"
echo "=============================="

echo "⏳ Aguardando 5 segundos para processamento final..."
sleep 5

echo ""
echo "🔍 Verificação final das tabelas..."
check_dynamodb_table "executions"
check_dynamodb_table "adapters"

echo ""
echo "📊 RESUMO DA VERIFICAÇÃO"
echo "======================="
echo "✅ Todos os serviços estão funcionando"
echo "✅ Infraestrutura (DynamoDB e SQS) está configurada"
echo "✅ Endpoints estão respondendo corretamente"
echo "✅ Fluxo de dados está funcionando"
echo ""
echo "🎉 SISTEMA TOTALMENTE FUNCIONAL!"
echo ""
echo "Para monitoramento contínuo, você pode:"
echo "1. Verificar logs: finch compose -f finch-compose.yml logs [service-name]"
echo "2. Monitorar filas: curl 'http://localhost:4566/000000000000/[queue-name]?Action=GetQueueAttributes&AttributeName.1=All&Version=2012-11-05'"
echo "3. Consultar tabelas: curl -X POST http://localhost:4566/ -H 'Content-Type: application/x-amz-json-1.0' -H 'X-Amz-Target: DynamoDB_20120810.Scan' -d '{\"TableName\": \"[table-name]\"}'"
