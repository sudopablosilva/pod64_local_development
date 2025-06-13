#!/bin/bash

echo "=== Testing Complete Flow ==="

# Generate random execution names to avoid key collisions
TIMESTAMP=$(date +%s)
RANDOM_ID=$(openssl rand -hex 4)
EXECUTION_NAME_1="TEST_${TIMESTAMP}_${RANDOM_ID}"
EXECUTION_NAME_2="SYNTH_${TIMESTAMP}_${RANDOM_ID}"
EXECUTION_NAME_3="JMW_${TIMESTAMP}_${RANDOM_ID}"

echo "🎯 Using random execution names:"
echo "   • Primary: $EXECUTION_NAME_1"
echo "   • Synthetic: $EXECUTION_NAME_2"
echo "   • JMW: $EXECUTION_NAME_3"
echo ""

# Test 1: Start Integrator (JMI)
echo "1. Testing Start Integrator (JMI)..."
EXECUTION_UUID=$(curl -s -X POST http://localhost:4333/startExecution \
-H "Content-Type: application/json" \
-d "{
    \"executionName\": \"$EXECUTION_NAME_1\"
}" | jq -r '.executionUuid // empty')

if [ -n "$EXECUTION_UUID" ]; then
    echo "✓ JMI Start Execution successful. UUID: $EXECUTION_UUID"
else
    echo "✗ JMI Start Execution failed"
fi

# Test 2: Start Integrator - Synthetic Test
echo "2. Testing Start Integrator - Synthetic Test..."
curl -s -X POST http://localhost:4333/startExecution \
-H "Content-Type: application/json" \
-d "{
    \"executionName\": \"$EXECUTION_NAME_2\"
}" > /dev/null

if [ $? -eq 0 ]; then
    echo "✓ JMI Synthetic Test successful"
else
    echo "✗ JMI Synthetic Test failed"
fi

# Test 3: JMW Start (from startRoutine.sh)
echo "3. Testing JMW Start..."
JMW_UUID=$(curl -s -X POST http://localhost:8080/start \
-H "Content-Type: application/json" \
-d "{
    \"executionName\": \"$EXECUTION_NAME_3\",
    \"accountId\": \"017820684888\",
    \"commonProperties\": {
        \"accountId\": \"017820684888\",
        \"providerConfigRef\": \"\",
        \"region\": \"sa-east-1\",
        \"tags\": {
            \"proprietario-equipe-e-mail\": \"douglas.pinheiro-santos@itau-unibanco.com.br\",
            \"tech-team-email\": \"ItauBatchOnCloud@itau-unibanco.com.br\"
        }
    },
    \"runtimes\": [
        {
            \"compute\": {
                \"type\": \"sampleruntime\",
                \"description\": \"Job de carregamento de works e temps\"
            },
            \"runtimeName\": \"jp7w\",
            \"security\": {
                \"cloudwatchEncryptionMode\": \"SSE-KMS\",
                \"executionRoleArn\": \"arn:aws:iam::017820684888:role/iamsr/role-glue-test-simples-iamsr\"
            },
            \"tags\": {
                \"valor1\": \"xpto\"
            }
        }
    ],
    \"schedulerRoutine\": {
        \"executionName\": \"JP799999\",
        \"cron\": \"0 6 * * *\",
        \"dependsOn\": \"None\",
        \"priority\": \"medium\",
        \"provisioning\": \"manual\",
        \"steps\": [
            {
                \"stepId\": \"JP799999-1\",
                \"tasks\": [
                    {
                        \"taskId\": \"JP7F001G\",
                        \"runtimeName\": \"jp7f\",
                        \"parameters\": {
                            \"PARM1\": \"estrategiasrecupcredito-jobtempfinalclie\"
                        }
                    }
                ]
            }
        ]
    }
}")

if [ -n "$JMW_UUID" ]; then
    echo "✓ JMW Start successful. UUID: $JMW_UUID"
else
    echo "✗ JMW Start failed"
fi

# Test 4: SPA Trigger
echo "4. Testing SPA Trigger..."
curl -s -X POST http://localhost:4446/v1/trigger \
-H "Content-Type: application/json" \
-d "{
    \"accountId\": \"017820684888\",
    \"executionName\": \"$EXECUTION_NAME_3\",
    \"eventDate\": \"2025-06-10T14:48:00Z\",
    \"eventType\": \"ForceJob\",
    \"eventId\": \"ID1025121314151\",
    \"parameters\": {
        \"key1\": \"value1\",
        \"key2\": \"value2\"
    }
}" > /dev/null

if [ $? -eq 0 ]; then
    echo "✓ SPA Trigger successful"
else
    echo "✗ SPA Trigger failed"
fi

# Test 5: SPA Schedule Creation
echo "5. Testing SPA Schedule Creation..."
SCHEDULE_NAME="DEMO_ROTINA_${RANDOM_ID}"
curl -s -X POST http://localhost:4444/v1/schedule \
-H "Content-Type: application/json" \
-d "{
    \"acronym\": \"A5\",
    \"repo\": \"BOC_DEMO_FOLDER\",
    \"routines\": [
        {
            \"name\": \"$SCHEDULE_NAME\",
            \"description\": \"Teste rotina\",
            \"cron\": \"0 17-21 * * 1,3,5\",
            \"priority\": \"high\",
            \"dependsOn\": [\"dependencia_1\", \"dependencia_2\"]
        }
    ]
}" > /dev/null

if [ $? -eq 0 ]; then
    echo "✓ SPA Schedule Creation successful"
else
    echo "✗ SPA Schedule Creation failed"
fi

# Test 6: Stop Execution
if [ -n "$EXECUTION_UUID" ]; then
    echo "6. Testing Stop Execution..."
    curl -s -X POST http://localhost:4333/stopExecution \
    -H "Content-Type: application/json" \
    -d "{
        \"executionName\": \"$EXECUTION_NAME_1\",
        \"executionUuid\": \"$EXECUTION_UUID\"
    }" > /dev/null

    if [ $? -eq 0 ]; then
        echo "✓ Stop Execution successful"
    else
        echo "✗ Stop Execution failed"
    fi
fi

# Test 7: Health checks
echo "7. Testing Health Checks..."
for port in 4333 8080 8084 8085 4444 8087; do
    response=$(curl -s http://localhost:$port/health)
    if [ $? -eq 0 ]; then
        echo "✓ Service on port $port is healthy"
    else
        echo "✗ Service on port $port is not responding"
    fi
done

echo "=== Flow Testing Complete ==="
echo ""
echo "📊 VERIFICANDO DADOS CRIADOS (via microserviços):"
echo "================================================="
echo ""
echo "1. Execuções na tabela (via JMI):"
curl -s http://localhost:4333/executions | jq '{count: .count, sample_executions: .executions[0:3] | map({name: .executionName, stage: .stage, status: .status})}'
echo ""
echo "2. Tabelas disponíveis (via JMI):"
curl -s http://localhost:4333/tables | jq '{count: .count, tables: .tables}'
echo ""
echo "3. Filas SQS (via JMI):"
curl -s http://localhost:4333/queues | jq '{count: .count, queues: .queues | map({name: .name, visible: .visibleMessages, processing: .notVisibleMessages})}'
echo ""
echo "💡 COMANDOS ÚTEIS PARA MONITORAMENTO:"
echo "===================================="
echo "• ./dashboard.sh                      - Dashboard em tempo real"
echo "• curl http://localhost:4333/executions - Ver todas as execuções"
echo "• curl http://localhost:4333/tables     - Listar tabelas DynamoDB"
echo "• curl http://localhost:4333/queues     - Ver status das filas SQS"
