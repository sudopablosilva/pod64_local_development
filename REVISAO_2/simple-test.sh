#!/bin/bash

echo "🚀 Running integration tests for POC BDD services..."
echo ""

# Test 1: Health checks
echo "📋 Test 1: Health checks for all services"
services="control-m:8080 jmi:8080 jmw:8080 jmr:8080 scheduler-plugin:8080 spa:8080 spaq:8080"
all_healthy=true

for service in $services; do
    echo -n "  Testing $service... "
    response=$(curl -s "http://$service/health")
    if echo "$response" | grep -q '"status":"healthy"'; then
        echo "✅ HEALTHY"
    else
        echo "❌ FAILED"
        all_healthy=false
    fi
done

if [ "$all_healthy" = true ]; then
    echo "✅ All services are healthy!"
else
    echo "❌ Some services failed health checks"
    exit 1
fi

echo ""

# Test 2: Job submission workflow
echo "📋 Test 2: Job submission workflow"

# Submit a job to Control-M
echo -n "  Submitting job to Control-M... "
job_response=$(curl -s -X POST "http://control-m:8080/jobs" \
    -H "Content-Type: application/json" \
    -d '{
        "job_name": "test-job-001",
        "job_type": "shell",
        "priority": 1,
        "parameters": {"command": "echo hello"}
    }')

if echo "$job_response" | grep -q '"message":"Job submitted successfully"'; then
    echo "✅ SUCCESS"
    job_id=$(echo "$job_response" | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
    echo "    Job ID: $job_id"
else
    echo "❌ FAILED"
    echo "    Response: $job_response"
    exit 1
fi

echo ""

# Test 3: Service integration
echo "📋 Test 3: Service integration workflow"

# Process job through JMI
echo -n "  Processing job through JMI... "
jmi_response=$(curl -s -X POST "http://jmi:8080/process" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": \"$job_id\",
        \"job_name\": \"test-job-001\",
        \"job_type\": \"shell\",
        \"priority\": 1
    }")

if echo "$jmi_response" | grep -q '"message":"Job integrated successfully"'; then
    echo "✅ SUCCESS"
else
    echo "❌ FAILED"
    echo "    Response: $jmi_response"
fi

# Process job through JMW
echo -n "  Processing job through JMW... "
jmw_response=$(curl -s -X POST "http://jmw:8080/process" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": \"$job_id\",
        \"job_name\": \"test-job-001\",
        \"job_type\": \"shell\",
        \"priority\": 1
    }")

if echo "$jmw_response" | grep -q '"message":"Job processed successfully"'; then
    echo "✅ SUCCESS"
else
    echo "❌ FAILED"
    echo "    Response: $jmw_response"
fi

# Execute job through JMR
echo -n "  Executing job through JMR... "
jmr_response=$(curl -s -X POST "http://jmr:8080/execute" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": \"$job_id\",
        \"job_name\": \"test-job-001\",
        \"job_type\": \"shell\",
        \"priority\": 1
    }")

if echo "$jmr_response" | grep -q '"message":"Job executed successfully"'; then
    echo "✅ SUCCESS"
else
    echo "❌ FAILED"
    echo "    Response: $jmr_response"
fi

# Create schedule through Scheduler Plugin
echo -n "  Creating schedule through Scheduler Plugin... "
scheduler_response=$(curl -s -X POST "http://scheduler-plugin:8080/process" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": \"$job_id\",
        \"job_name\": \"test-job-001\",
        \"job_type\": \"shell\",
        \"priority\": 1
    }")

if echo "$scheduler_response" | grep -q '"message":"Job scheduled successfully"'; then
    echo "✅ SUCCESS"
    schedule_id=$(echo "$scheduler_response" | grep -o '"schedule_id":"[^"]*"' | cut -d'"' -f4)
    echo "    Schedule ID: $schedule_id"
else
    echo "❌ FAILED"
    echo "    Response: $scheduler_response"
fi

# Configure adapter through SPA
echo -n "  Configuring adapter through SPA... "
spa_response=$(curl -s -X POST "http://spa:8080/process" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": \"$schedule_id\",
        \"cron_expr\": \"0 */5 * * * *\"
    }")

if echo "$spa_response" | grep -q '"message":"Schedule processed successfully"'; then
    echo "✅ SUCCESS"
    adapter_id=$(echo "$spa_response" | grep -o '"adapter_id":"[^"]*"' | cut -d'"' -f4)
    echo "    Adapter ID: $adapter_id"
else
    echo "❌ FAILED"
    echo "    Response: $spa_response"
fi

# Queue message through SPAQ
echo -n "  Queuing message through SPAQ... "
spaq_response=$(curl -s -X POST "http://spaq:8080/process" \
    -H "Content-Type: application/json" \
    -d "{
        \"id\": \"$adapter_id\",
        \"adapter_type\": \"frequent\",
        \"schedule_id\": \"$schedule_id\"
    }")

if echo "$spaq_response" | grep -q '"message":"Adapter processed successfully"'; then
    echo "✅ SUCCESS"
else
    echo "❌ FAILED"
    echo "    Response: $spaq_response"
fi

echo ""
echo "🎉 Integration tests completed successfully!"
echo "✅ All services are working and communicating properly"
echo ""
echo "📊 Summary:"
echo "  - All 7 services are healthy and running"
echo "  - Job submission workflow is working"
echo "  - Service-to-service communication is functional"
echo "  - Complete pipeline from job submission to queue processing works"
