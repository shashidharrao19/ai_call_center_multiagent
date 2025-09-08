#!/bin/bash

# AI Call Center Setup Script
# This script sets up the development environment

set -e

echo "🚀 Setting up AI Call Center..."

# Check if required tools are installed
check_requirements() {
    echo "📋 Checking requirements..."
    
    if ! command -v go &> /dev/null; then
        echo "❌ Go is not installed. Please install Go 1.21+"
        exit 1
    fi
    
    if ! command -v python3 &> /dev/null; then
        echo "❌ Python 3 is not installed. Please install Python 3.11+"
        exit 1
    fi
    
    if ! command -v node &> /dev/null; then
        echo "❌ Node.js is not installed. Please install Node.js 18+"
        exit 1
    fi
    
    echo "✅ All requirements met"
}

# Setup environment
setup_environment() {
    echo "🔧 Setting up environment..."
    
    # Copy environment file if it doesn't exist
    if [ ! -f .env ]; then
        cp .env.example .env
        echo "📝 Created .env file from .env.example"
        echo "⚠️  Please edit .env file with your API keys"
    fi
    
    echo "✅ Environment setup complete"
}

# Setup Go backend
setup_backend() {
    echo "🔧 Setting up Go backend..."
    
    cd backend
    
    # Initialize Go module if needed
    if [ ! -f go.mod ]; then
        go mod init ai-call-center/backend
    fi
    
    # Download dependencies
    go mod tidy
    
    echo "✅ Go backend setup complete"
    cd ..
}

# Setup Python AI engine
setup_ai_engine() {
    echo "🔧 Setting up Python AI engine..."
    
    cd ai-engine
    
    # Create virtual environment if it doesn't exist
    if [ ! -d "venv" ]; then
        python3 -m venv venv
    fi
    
    # Activate virtual environment
    source venv/bin/activate
    
    # Install dependencies
    pip install --upgrade pip
    pip install -r requirements.txt
    
    echo "✅ Python AI engine setup complete"
    cd ..
}

# Setup frontend
setup_frontend() {
    echo "🔧 Setting up frontend..."
    
    cd frontend
    
    # Install dependencies
    npm install
    
    echo "✅ Frontend setup complete"
    cd ..
}

# Create necessary directories
create_directories() {
    echo "📁 Creating necessary directories..."
    
    mkdir -p logs
    mkdir -p data
    mkdir -p temp
    
    echo "✅ Directories created"
}

# Main setup function
main() {
    echo "🎯 Starting AI Call Center setup..."
    
    check_requirements
    setup_environment
    create_directories
    setup_backend
    setup_ai_engine
    setup_frontend
    
    echo ""
    echo "🎉 Setup complete!"
    echo ""
    echo "Next steps:"
    echo "1. Edit .env file with your Google API key"
    echo "2. Run: ./scripts/start.sh"
    echo "3. Open http://localhost:3000 in your browser"
    echo ""
    echo "For development:"
    echo "- Backend: cd backend && go run main.go"
    echo "- AI Engine: cd ai-engine && source venv/bin/activate && python main.py"
    echo "- Frontend: cd frontend && npm run dev"
}

# Run main function
main "$@"
