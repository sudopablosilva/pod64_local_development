#!/bin/bash

# Selenium Test Runner for POC BDD Web Dashboard
# This script sets up the environment and runs comprehensive tests

echo "🧪 POC BDD Web Dashboard - Selenium Test Suite"
echo "=============================================="

# Check if Python 3 is available
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is required but not installed"
    exit 1
fi

# Check if the dashboard is running
echo "🔍 Checking if dashboard is running..."
if ! curl -s http://localhost:3000/health > /dev/null; then
    echo "❌ Dashboard is not running on port 3000"
    echo "Please start the dashboard first:"
    echo "  ./start-web-dashboard.sh"
    exit 1
fi

echo "✅ Dashboard is running"

# Create virtual environment if it doesn't exist
if [ ! -d "venv" ]; then
    echo "📦 Creating Python virtual environment..."
    python3 -m venv venv
fi

# Activate virtual environment
echo "🔧 Activating virtual environment..."
source venv/bin/activate

# Install dependencies
echo "📥 Installing test dependencies..."
pip install -r requirements.txt

# Create reports directory
mkdir -p reports

# Run tests with different levels
echo ""
echo "🚀 Running Selenium Tests..."
echo "=============================================="

# Run basic tests first
echo "1️⃣ Running Basic Functionality Tests..."
python -m pytest test_dashboard_basic.py -v -m "not slow" --html=reports/basic_tests.html --self-contained-html

if [ $? -ne 0 ]; then
    echo "❌ Basic tests failed. Stopping test execution."
    exit 1
fi

echo ""
echo "2️⃣ Running Data Display Tests..."
python -m pytest test_dashboard_data.py -v --html=reports/data_tests.html --self-contained-html

echo ""
echo "3️⃣ Running Real-time Functionality Tests..."
python -m pytest test_dashboard_realtime.py -v --html=reports/realtime_tests.html --self-contained-html

echo ""
echo "4️⃣ Running Accessibility Tests..."
python -m pytest test_dashboard_accessibility.py -v --html=reports/accessibility_tests.html --self-contained-html

echo ""
echo "5️⃣ Running Complete Test Suite..."
python -m pytest . -v --html=reports/complete_test_report.html --self-contained-html

echo ""
echo "📊 Test Results Summary:"
echo "=============================================="
echo "📁 Test reports saved in: ./reports/"
echo "🌐 Open reports/complete_test_report.html in your browser"
echo ""

# Check if any tests failed
if [ $? -eq 0 ]; then
    echo "✅ All tests completed successfully!"
    echo ""
    echo "📋 Next Steps:"
    echo "• Review test reports in ./reports/ directory"
    echo "• Check dashboard functionality at http://localhost:3000"
    echo "• Run individual test files for specific issues"
else
    echo "⚠️  Some tests may have failed. Check the reports for details."
    echo ""
    echo "🔧 Troubleshooting:"
    echo "• Ensure all services are running: ./start-web-dashboard.sh"
    echo "• Check browser compatibility (Chrome required)"
    echo "• Verify network connectivity to localhost:3000"
fi

# Deactivate virtual environment
deactivate

echo ""
echo "🏁 Test execution completed!"
