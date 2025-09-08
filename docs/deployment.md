# AI Call Center Deployment Guide

## Overview

This guide covers deployment options for the AI Call Center, from local development to production cloud deployment.

## Prerequisites

### Required Software
- **Go 1.21+**: For backend service
- **Python 3.11+**: For AI engine
- **Node.js 18+**: For frontend
- **Docker & Docker Compose**: For containerized deployment
- **Google Cloud SDK**: For GCP deployment (optional)

### Required Accounts
- **Google Cloud Platform**: For Gemini API access
- **API Keys**: Google API key with Gemini access

## Local Development

### 1. Quick Start

```bash
# Clone the repository
git clone <repository-url>
cd ai-call-center

# Run setup script
./scripts/setup.sh

# Edit environment variables
nano .env

# Start all services
./scripts/start.sh
```

### 2. Manual Setup

#### Backend (Go)
```bash
cd backend
go mod tidy
go run main.go
```

#### AI Engine (Python)
```bash
cd ai-engine
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
python main.py
```

#### Frontend (JavaScript)
```bash
cd frontend
npm install
npm run dev
```

### 3. Environment Configuration

Create `.env` file:
```bash
# Gemini API
GOOGLE_API_KEY=your_gemini_api_key_here
GEMINI_MODEL=models/gemini-2.0-flash-live-001

# Go Backend
GO_PORT=8080
GO_WEBSOCKET_PATH=/ws
GO_MAX_CONNECTIONS=1000

# Python AI Engine
PYTHON_PORT=8000
MCP_SERVER_URL=http://localhost:8001

# Frontend
FRONTEND_PORT=3000
WEBSOCKET_URL=ws://localhost:8080/ws
```

## Docker Deployment

### 1. Build and Run

```bash
# Build all services
docker-compose build

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### 2. Individual Service Deployment

#### Backend Dockerfile
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
```

#### AI Engine Dockerfile
```dockerfile
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["python", "main.py"]
```

#### Frontend Dockerfile
```dockerfile
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

## Cloud Deployment

### Google Cloud Platform

#### 1. Setup GCP Project

```bash
# Install Google Cloud SDK
curl https://sdk.cloud.google.com | bash
exec -l $SHELL

# Initialize and authenticate
gcloud init
gcloud auth login

# Create project
gcloud projects create ai-call-center-$(date +%s)
gcloud config set project ai-call-center-$(date +%s)

# Enable required APIs
gcloud services enable run.googleapis.com
gcloud services enable cloudbuild.googleapis.com
```

#### 2. Deploy Backend (Cloud Run)

```bash
cd backend

# Build and deploy
gcloud run deploy ai-call-center-backend \
  --source . \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080 \
  --memory 1Gi \
  --cpu 1 \
  --max-instances 10
```

#### 3. Deploy AI Engine (Cloud Run)

```bash
cd ai-engine

# Build and deploy
gcloud run deploy ai-call-center-engine \
  --source . \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8000 \
  --memory 2Gi \
  --cpu 2 \
  --max-instances 5 \
  --set-env-vars GOOGLE_API_KEY=$GOOGLE_API_KEY
```

#### 4. Deploy Frontend (Cloud Storage)

```bash
cd frontend

# Build frontend
npm run build

# Create bucket
gsutil mb gs://ai-call-center-frontend

# Upload files
gsutil -m cp -r dist/* gs://ai-call-center-frontend

# Make bucket public
gsutil iam ch allUsers:objectViewer gs://ai-call-center-frontend
```

### AWS Deployment

#### 1. Deploy with ECS

```bash
# Create ECS cluster
aws ecs create-cluster --cluster-name ai-call-center

# Create task definitions for each service
aws ecs register-task-definition --cli-input-json file://backend-task-definition.json
aws ecs register-task-definition --cli-input-json file://ai-engine-task-definition.json

# Create services
aws ecs create-service \
  --cluster ai-call-center \
  --service-name backend \
  --task-definition backend:1 \
  --desired-count 2
```

#### 2. Deploy Frontend (S3 + CloudFront)

```bash
# Create S3 bucket
aws s3 mb s3://ai-call-center-frontend

# Upload files
aws s3 sync dist/ s3://ai-call-center-frontend

# Create CloudFront distribution
aws cloudfront create-distribution --distribution-config file://cloudfront-config.json
```

## Production Configuration

### 1. Environment Variables

```bash
# Production environment
NODE_ENV=production
LOG_LEVEL=warn
LOG_FORMAT=json

# Security
CORS_ORIGINS=https://yourdomain.com
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60

# Performance
MAX_CONNECTIONS=1000
CONNECTION_TIMEOUT=30
AUDIO_BUFFER_SIZE=4096
```

### 2. Load Balancer Configuration

#### Nginx Configuration
```nginx
upstream backend {
    server backend1:8080;
    server backend2:8080;
    server backend3:8080;
}

upstream ai-engine {
    server ai-engine1:8000;
    server ai-engine2:8000;
}

server {
    listen 80;
    server_name yourdomain.com;

    # WebSocket proxy
    location /ws {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # API proxy
    location /api/ {
        proxy_pass http://ai-engine/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Static files
    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }
}
```

### 3. Monitoring Setup

#### Prometheus Configuration
```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'ai-call-center-backend'
    static_configs:
      - targets: ['backend:8080']
    metrics_path: /metrics

  - job_name: 'ai-call-center-ai-engine'
    static_configs:
      - targets: ['ai-engine:8000']
    metrics_path: /metrics
```

#### Grafana Dashboard
Import the provided Grafana dashboard configuration for monitoring:
- Connection metrics
- Audio quality metrics
- Response time distribution
- Error rates

## Scaling

### Horizontal Scaling

```bash
# Scale backend services
docker-compose up -d --scale backend=3

# Scale AI engine services
docker-compose up -d --scale ai-engine=2
```

### Auto-scaling (GCP Cloud Run)

```bash
# Configure auto-scaling
gcloud run services update ai-call-center-backend \
  --min-instances 1 \
  --max-instances 10 \
  --cpu-throttling \
  --concurrency 100
```

## Security

### 1. SSL/TLS Configuration

```bash
# Generate SSL certificate
certbot certonly --webroot -w /var/www/html -d yourdomain.com

# Update nginx configuration
server {
    listen 443 ssl;
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
}
```

### 2. Firewall Configuration

```bash
# Allow only necessary ports
ufw allow 22    # SSH
ufw allow 80    # HTTP
ufw allow 443   # HTTPS
ufw enable
```

### 3. API Key Management

```bash
# Use Google Secret Manager
gcloud secrets create google-api-key --data-file=api-key.txt

# Access in Cloud Run
gcloud run services update ai-call-center-engine \
  --set-secrets GOOGLE_API_KEY=google-api-key:latest
```

## Troubleshooting

### Common Issues

1. **WebSocket Connection Failed**
   - Check firewall settings
   - Verify WebSocket URL
   - Check CORS configuration

2. **Audio Quality Issues**
   - Verify sample rate configuration
   - Check network latency
   - Monitor audio buffer sizes

3. **High Memory Usage**
   - Check for memory leaks
   - Optimize audio processing
   - Monitor session cleanup

### Log Analysis

```bash
# View real-time logs
docker-compose logs -f

# Filter logs by service
docker-compose logs -f backend
docker-compose logs -f ai-engine

# Search for errors
docker-compose logs | grep ERROR
```

### Performance Monitoring

```bash
# Monitor resource usage
docker stats

# Check connection count
curl http://localhost:8080/health

# Monitor response times
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8000/health
```

## Backup and Recovery

### Database Backup
```bash
# Backup PostgreSQL
pg_dump ai_call_center > backup.sql

# Restore database
psql ai_call_center < backup.sql
```

### Configuration Backup
```bash
# Backup configuration files
tar -czf config-backup.tar.gz .env docker-compose.yml
```

## Maintenance

### Regular Tasks
- Monitor disk space
- Update dependencies
- Review security patches
- Analyze performance metrics
- Clean up old sessions

### Update Procedure
```bash
# Pull latest changes
git pull origin main

# Rebuild and restart services
docker-compose down
docker-compose build
docker-compose up -d

# Verify deployment
./scripts/test.sh
```
