#!/bin/bash

# POC BDD Web Dashboard Startup Script
# Starts the modern web dashboard with UX best practices

echo "🚀 Starting POC BDD Web Dashboard..."
echo "=============================================="

# Check if finch is available
if ! command -v finch &> /dev/null; then
    echo "❌ Finch is not installed or not in PATH"
    echo "Please install Finch: brew install finch"
    exit 1
fi

# Check if the web-dashboard directory exists
if [ ! -d "web-dashboard" ]; then
    echo "❌ Web dashboard directory not found"
    echo "Please ensure you're in the project root directory"
    exit 1
fi

# Function to check if a port is in use
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0  # Port is in use
    else
        return 1  # Port is free
    fi
}

# Check if port 3000 is available
if check_port 3000; then
    echo "⚠️  Port 3000 is already in use"
    echo "Please stop the service using port 3000 or use a different port"
    echo "Current process using port 3000:"
    lsof -Pi :3000 -sTCP:LISTEN
    exit 1
fi

# Start the complete system with web dashboard
echo "📦 Starting all services including web dashboard..."
finch compose -f finch-compose.yml up -d

# Wait for services to start
echo "⏳ Waiting for services to initialize..."
sleep 30

# Check service health
echo ""
echo "🏥 Checking service health..."
echo "=============================================="

services=(
    "LocalStack:4566"
    "Web Dashboard:3000"
    "JMI:4333"
    "JMW:8080"
    "JMR:8084"
    "Scheduler:8085"
    "SPA:4444"
    "SPAQ:8087"
)

all_healthy=true

for service_info in "${services[@]}"; do
    service_name=$(echo "$service_info" | cut -d: -f1)
    service_port=$(echo "$service_info" | cut -d: -f2)
    
    if [ "$service_name" = "LocalStack" ]; then
        # Special check for LocalStack
        if curl -s http://localhost:$service_port/health > /dev/null 2>&1; then
            echo "✅ $service_name (port $service_port): Online"
        else
            echo "❌ $service_name (port $service_port): Offline"
            all_healthy=false
        fi
    else
        # Standard health check for other services
        if curl -s http://localhost:$service_port/health > /dev/null 2>&1; then
            echo "✅ $service_name (port $service_port): Online"
        else
            echo "❌ $service_name (port $service_port): Offline"
            all_healthy=false
        fi
    fi
done

echo ""
if [ "$all_healthy" = true ]; then
    echo "🎉 All services are healthy!"
else
    echo "⚠️  Some services are not responding. They may still be starting up."
    echo "Wait a few more minutes and check again."
fi

echo ""
echo "🌐 Web Dashboard Access:"
echo "=============================================="
echo "📊 Modern Dashboard: http://localhost:3000"
echo "🔧 Dashboard API: http://localhost:3000/api/dashboard"
echo "💓 Health Check: http://localhost:3000/health"
echo ""
echo "📱 Features Available:"
echo "• Real-time monitoring with WebSocket updates"
echo "• Responsive design for mobile and desktop"
echo "• Interactive service health monitoring"
echo "• Advanced filtering and search capabilities"
echo "• Direct links to LocalStack admin interfaces"
echo "• Accessibility compliant with keyboard navigation"
echo ""
echo "🔗 Alternative Access:"
echo "• Legacy Dashboard: ./dashboard.sh"
echo "• Direct API Access: curl http://localhost:4333/executions"
echo ""
echo "📋 Quick Commands:"
echo "• View logs: finch compose -f finch-compose.yml logs web-dashboard"
echo "• Restart dashboard: finch compose -f finch-compose.yml restart web-dashboard"
echo "• Stop all services: finch compose -f finch-compose.yml down"
echo ""
echo "🧪 Testing:"
echo "• Run complete test: ./test-complete-flow.sh"
echo "• Test single execution: curl -X POST http://localhost:4333/startExecution -H 'Content-Type: application/json' -d '{\"executionName\": \"WEB_TEST_001\"}'"
echo ""

# Open the dashboard in the default browser (macOS)
if command -v open &> /dev/null; then
    echo "🌐 Opening dashboard in your default browser..."
    sleep 2
    open http://localhost:3000
fi

echo "✨ Web Dashboard is ready! Enjoy the modern monitoring experience."
