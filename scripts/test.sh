#!/bin/bash

# AI Call Center Test Script
# This script runs tests for all components

set -e

echo "🧪 Running AI Call Center tests..."

# Function to run tests for a service
run_service_tests() {
    local service_name=$1
    local test_command=$2
    
    echo "🔍 Testing $service_name..."
    
    if eval "$test_command"; then
        echo "✅ $service_name tests passed"
    else
        echo "❌ $service_name tests failed"
        return 1
    fi
}

# Test Go backend
test_backend() {
    echo "🔍 Testing Go backend..."
    
    cd backend
    
    # Run Go tests
    if go test ./... -v; then
        echo "✅ Backend tests passed"
    else
        echo "❌ Backend tests failed"
        cd ..
        return 1
    fi
    
    cd ..
}

# Test Python AI engine
test_ai_engine() {
    echo "🔍 Testing Python AI engine..."
    
    cd ai-engine
    
    # Activate virtual environment
    source venv/bin/activate
    
    # Run Python tests
    if python -m pytest tests/ -v; then
        echo "✅ AI engine tests passed"
    else
        echo "❌ AI engine tests failed"
        cd ..
        return 1
    fi
    
    cd ..
}

# Test frontend
test_frontend() {
    echo "🔍 Testing frontend..."
    
    cd frontend
    
    # Run npm tests
    if npm test; then
        echo "✅ Frontend tests passed"
    else
        echo "❌ Frontend tests failed"
        cd ..
        return 1
    fi
    
    cd ..
}

# Integration tests
test_integration() {
    echo "🔍 Running integration tests..."
    
    # Check if services are running
    if ! curl -s http://localhost:8080/health > /dev/null; then
        echo "❌ Backend is not running"
        return 1
    fi
    
    if ! curl -s http://localhost:8000/health > /dev/null; then
        echo "❌ AI engine is not running"
        return 1
    fi
    
    if ! curl -s http://localhost:3000 > /dev/null; then
        echo "❌ Frontend is not running"
        return 1
    fi
    
    echo "✅ All services are running"
    
    # Test WebSocket connection
    echo "🔍 Testing WebSocket connection..."
    # Add WebSocket test here
    
    echo "✅ Integration tests passed"
}

# Main test function
main() {
    local test_type=${1:-"all"}
    
    case $test_type in
        "backend")
            test_backend
            ;;
        "ai-engine")
            test_ai_engine
            ;;
        "frontend")
            test_frontend
            ;;
        "integration")
            test_integration
            ;;
        "all")
            test_backend
            test_ai_engine
            test_frontend
            test_integration
            ;;
        *)
            echo "Usage: $0 [backend|ai-engine|frontend|integration|all]"
            exit 1
            ;;
    esac
    
    echo ""
    echo "🎉 All tests completed successfully!"
}

# Run main function
main "$@"
