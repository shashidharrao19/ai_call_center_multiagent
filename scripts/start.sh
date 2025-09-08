#!/bin/bash

# AI Call Center Start Script
# This script starts all services

set -e

echo "🚀 Starting AI Call Center..."

# Check if .env file exists
if [ ! -f .env ]; then
    echo "❌ .env file not found. Please run ./scripts/setup.sh first"
    exit 1
fi

# Load environment variables
source .env

# Function to start a service in background
start_service() {
    local service_name=$1
    local command=$2
    local log_file=$3
    
    echo "🔄 Starting $service_name..."
    
    # Create logs directory if it doesn't exist
    mkdir -p logs
    
    # Start service in background
    nohup $command > "logs/${log_file}" 2>&1 &
    local pid=$!
    
    # Save PID for cleanup
    echo $pid > "logs/${service_name}.pid"
    
    echo "✅ $service_name started (PID: $pid)"
}

# Start services
start_services() {
    echo "🔧 Starting services..."
    
    # Start Go backend
    cd backend
    start_service "backend" "go run main.go" "backend.log"
    cd ..
    
    # Start Python AI engine
    cd ai-engine
    start_service "ai-engine" "source venv/bin/activate && python main.py" "ai-engine.log"
    cd ..
    
    # Start frontend
    cd frontend
    start_service "frontend" "npm run dev" "frontend.log"
    cd ..
    
    echo "✅ All services started"
}

# Wait for services to be ready
wait_for_services() {
    echo "⏳ Waiting for services to be ready..."
    
    # Wait for backend
    echo "Waiting for backend (port 8080)..."
    while ! curl -s http://localhost:8080/health > /dev/null; do
        sleep 1
    done
    echo "✅ Backend is ready"
    
    # Wait for AI engine
    echo "Waiting for AI engine (port 8000)..."
    while ! curl -s http://localhost:8000/health > /dev/null; do
        sleep 1
    done
    echo "✅ AI engine is ready"
    
    # Wait for frontend
    echo "Waiting for frontend (port 3000)..."
    while ! curl -s http://localhost:3000 > /dev/null; do
        sleep 1
    done
    echo "✅ Frontend is ready"
}

# Show status
show_status() {
    echo ""
    echo "🎉 AI Call Center is running!"
    echo ""
    echo "Services:"
    echo "- Backend: http://localhost:8080"
    echo "- AI Engine: http://localhost:8000"
    echo "- Frontend: http://localhost:3000"
    echo ""
    echo "Logs:"
    echo "- Backend: logs/backend.log"
    echo "- AI Engine: logs/ai-engine.log"
    echo "- Frontend: logs/frontend.log"
    echo ""
    echo "To stop all services: ./scripts/stop.sh"
    echo "To view logs: tail -f logs/*.log"
}

# Main function
main() {
    start_services
    wait_for_services
    show_status
}

# Handle Ctrl+C
trap 'echo ""; echo "🛑 Stopping services..."; ./scripts/stop.sh; exit 0' INT

# Run main function
main "$@"
