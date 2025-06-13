#!/bin/bash

echo "=== Manual Testing Guide ==="
echo "Este script te guiará através de testes manuais do sistema"
echo ""

# Function to wait for user input
wait_for_user() {
    echo "Pressione ENTER para continuar..."
    read
}

# Function to make a request and show response
test_endpoint() {
    local method=$1
    local url=$2
    local data=$3
    local description=$4
    
    echo "=== $description ==="
    echo "Método: $method"
    echo "URL: $url"
    if [ -n "$data" ]; then
        echo "Payload:"
        echo "$data" | jq .
    fi
    echo ""
    echo "Executando requisição..."
    
    if [ -n "$data" ]; then
        response=$(curl -s -X $method "$url" -H "Content-Type: application/json" -d "$data")
    else
        response=$(curl -s -X $method "$url")
    fi
    
    echo "Resposta:"
    echo "$response" | jq . 2>/dev/null || echo "$response"
    echo ""
    wait_for_user
}

echo "1. Testando Health Checks de todos os serviços..."
wait_for_user

for port in 4333 8080 8084 8085 4444 8087; do
    test_endpoint "GET" "http://localhost:$port/health" "" "Health Check - Porta $port"
done

echo "2. Testando JMI - Start Execution..."
wait_for_user

EXECUTION_DATA='{
    "executionName": "TEST_EXECUTION_001"
}'

test_endpoint "POST" "http://localhost:4333/startExecution" "$EXECUTION_DATA" "JMI Start Execution"

echo "3. Testando JMW - Start (Payload do startRoutine.sh)..."
wait_for_user

JMW_DATA='{
    "executionName": "TEST_JMW_001",
    "accountId": "017820684888",
    "commonProperties": {
        "accountId": "017820684888",
        "region": "sa-east-1",
        "tags": {
            "test": "manual-execution"
        }
    },
    "runtimes": [
        {
            "runtimeName": "test-runtime",
            "compute": {
                "type": "test",
                "description": "Teste manual"
            },
            "security": {
                "executionRoleArn": "arn:aws:iam::017820684888:role/test-role"
            }
        }
    ],
    "schedulerRoutine": {
        "executionName": "TEST_ROUTINE",
        "cron": "0 6 * * *",
        "priority": "medium",
        "steps": [
            {
                "stepId": "STEP-1",
                "tasks": [
                    {
                        "taskId": "TASK-1",
                        "runtimeName": "test-runtime",
                        "parameters": {
                            "param1": "value1"
                        }
                    }
                ]
            }
        ]
    }
}'

test_endpoint "POST" "http://localhost:8080/start" "$JMW_DATA" "JMW Start"

echo "4. Testando SPA - Trigger..."
wait_for_user

TRIGGER_DATA='{
    "accountId": "017820684888",
    "executionName": "TEST_TRIGGER_001",
    "eventDate": "2025-06-12T20:00:00Z",
    "eventType": "ManualTest",
    "eventId": "MANUAL_TEST_001",
    "parameters": {
        "testType": "manual",
        "user": "tester"
    }
}'

test_endpoint "POST" "http://localhost:4446/v1/trigger" "$TRIGGER_DATA" "SPA Trigger"

echo "5. Testando SPA - Schedule Creation..."
wait_for_user

SCHEDULE_DATA='{
    "acronym": "TEST",
    "repo": "MANUAL_TEST_REPO",
    "routines": [
        {
            "name": "MANUAL_TEST_ROUTINE",
            "description": "Rotina de teste manual",
            "cron": "0 12 * * *",
            "priority": "high",
            "dependsOn": ["dependency1"]
        }
    ]
}'

test_endpoint "POST" "http://localhost:4444/v1/schedule" "$SCHEDULE_DATA" "SPA Schedule Creation"

echo "6. Verificando dados armazenados no LocalStack..."
wait_for_user

echo "=== Verificando Filas SQS ==="
echo "Listando filas disponíveis:"
curl -s "http://localhost:4566/000000000000/?Action=ListQueues&Version=2012-11-05" | grep -o 'http://[^<]*' || echo "Erro ao listar filas"
echo ""

echo "=== Verificando Tabelas DynamoDB ==="
echo "Listando tabelas disponíveis:"
curl -s -X POST "http://localhost:4566/" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -H "X-Amz-Target: DynamoDB_20120810.ListTables" \
  -d '{}' | jq . 2>/dev/null || echo "Erro ao listar tabelas"
echo ""

wait_for_user

echo "7. Consultando dados específicos das tabelas..."
wait_for_user

echo "=== Dados da tabela 'executions' ==="
curl -s -X POST "http://localhost:4566/" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -H "X-Amz-Target: DynamoDB_20120810.Scan" \
  -d '{"TableName": "executions"}' | jq . 2>/dev/null || echo "Erro ao consultar tabela executions"
echo ""

wait_for_user

echo "=== Dados da tabela 'adapters' ==="
curl -s -X POST "http://localhost:4566/" \
  -H "Content-Type: application/x-amz-json-1.0" \
  -H "X-Amz-Target: DynamoDB_20120810.Scan" \
  -d '{"TableName": "adapters"}' | jq . 2>/dev/null || echo "Erro ao consultar tabela adapters"
echo ""

echo "=== Teste Manual Completo! ==="
echo "Você testou todos os endpoints principais e verificou o armazenamento de dados."
echo "Para mais detalhes, você pode:"
echo "1. Verificar os logs dos containers: finch compose -f finch-compose.yml logs [service-name]"
echo "2. Acessar o LocalStack diretamente: http://localhost:4566"
echo "3. Executar consultas específicas no DynamoDB"
echo "4. Monitorar as filas SQS"
