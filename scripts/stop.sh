#!/bin/bash

# AI Call Center Stop Script
# This script stops all running services

set -e

echo "🛑 Stopping AI Call Center services..."

# Function to stop a service
stop_service() {
    local service_name=$1
    local pid_file="logs/${service_name}.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            echo "🔄 Stopping $service_name (PID: $pid)..."
            kill "$pid"
            
            # Wait for process to stop
            local count=0
            while kill -0 "$pid" 2>/dev/null && [ $count -lt 10 ]; do
                sleep 1
                count=$((count + 1))
            done
            
            # Force kill if still running
            if kill -0 "$pid" 2>/dev/null; then
                echo "⚠️  Force stopping $service_name..."
                kill -9 "$pid"
            fi
            
            echo "✅ $service_name stopped"
        else
            echo "⚠️  $service_name was not running"
        fi
        rm -f "$pid_file"
    else
        echo "⚠️  No PID file found for $service_name"
    fi
}

# Stop all services
stop_services() {
    stop_service "backend"
    stop_service "ai-engine"
    stop_service "frontend"
}

# Clean up any remaining processes
cleanup_processes() {
    echo "🧹 Cleaning up remaining processes..."
    
    # Kill any remaining Go processes
    pkill -f "go run main.go" 2>/dev/null || true
    
    # Kill any remaining Python processes
    pkill -f "python main.py" 2>/dev/null || true
    
    # Kill any remaining Node processes
    pkill -f "npm run dev" 2>/dev/null || true
    
    echo "✅ Cleanup complete"
}

# Main function
main() {
    stop_services
    cleanup_processes
    
    echo ""
    echo "🎉 All services stopped"
    echo ""
    echo "To start again: ./scripts/start.sh"
}

# Run main function
main "$@"
