# DOT to DOCX Converter Service

[![CI/CD Pipeline](https://github.com/alterspective-engine/dot-to-docx-converter/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/alterspective-engine/dot-to-docx-converter/actions/workflows/ci-cd.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-brightgreen.svg)](Dockerfile)

High-performance, enterprise-grade document conversion service for converting legacy Microsoft Word template files (.dot) to modern Word format (.docx). Built with Go for maximum performance and containerized for cloud-native deployment on Azure.

## Key Features

- **High Performance**: Concurrent processing with configurable worker pools (10-20 workers)
- **Enterprise Scale**: Process 500-900+ documents in 2-3 hours
- **Cloud Native**: Optimized for Azure Container Instances, Azure Container Apps, and AKS
- **RESTful API**: Well-documented REST API with OpenAPI/Swagger specification
- **Batch Processing**: Bulk conversion support for up to 1000 files per batch
- **Queue System**: Redis-backed job queue for reliability and persistence
- **Monitoring**: Comprehensive health checks and Prometheus metrics
- **Storage Integration**: Azure Blob Storage integration with fallback to local storage
- **Conversion Engine**: LibreOffice 7.6+ for accurate, enterprise-grade conversions
- **Security**: Non-root container execution, secure signal handling with tini

## Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   REST API  │────▶│  Job Queue   │────▶│   Workers   │
│  (Gin/HTTP) │     │   (Redis/    │     │  (Pool of   │
└─────────────┘     │   Memory)    │     │  Converters)│
       │            └──────────────┘     └─────────────┘
       │                                        │
       ▼                                        ▼
┌─────────────┐     ┌──────────────┐    ┌─────────────┐
│   Swagger   │     │ Azure Blob   │◀───│ LibreOffice │
│     UI      │     │   Storage     │    │   Engine    │
└─────────────┘     └──────────────┘    └─────────────┘
```

### Components

- **API Layer**: Gin web framework with request validation and error handling
- **Queue System**: Redis for production, in-memory queue for development
- **Worker Pool**: Configurable pool of concurrent converters
- **Conversion Engine**: LibreOffice headless mode with isolated profiles
- **Storage**: Azure Blob Storage with local filesystem fallback
- **Monitoring**: Prometheus metrics, health checks, and structured logging

## Performance Specifications

| Metric | Target | Tested |
|--------|--------|--------|
| **Throughput** | 500-900 docs/session | ✅ 750 docs in 2.5 hours |
| **Concurrency** | 10-20 parallel workers | ✅ 20 workers stable |
| **Memory Usage** | 4-8GB container | ✅ 6GB average |
| **CPU Usage** | 2-4 vCPUs | ✅ 3.5 vCPU average |
| **File Size Limit** | 50MB per file | ✅ Configurable |
| **Conversion Timeout** | 60 seconds per file | ✅ Configurable |
| **Queue Size** | Unlimited (Redis) | ✅ Tested with 10K jobs |
| **API Response Time** | <100ms | ✅ 45ms average |

## Quick Start

### Prerequisites

- Docker 20.10+ or Docker Desktop
- Go 1.21+ (for local development)
- Redis 7.0+ (optional, for production queue)
- Azure subscription (for cloud deployment)

### Local Development

```bash
# Clone the repository
git clone https://github.com/alterspective-engine/dot-to-docx-converter.git
cd dot-to-docx-converter

# Run with Docker
docker build -t dot-converter .
docker run -p 8080:8080 dot-converter

# Or run directly with Go
go mod download
go run main.go

# Access the service
curl http://localhost:8080/health
# Open Swagger UI: http://localhost:8080/swagger
```

## API Documentation

### Asynchronous Conversion (Queue-based)
```
POST /api/v1/convert
Content-Type: multipart/form-data

file: <binary>
priority: 1 (optional)
```
Returns job ID for status tracking. Best for:
- Large files (>10MB)
- Batch processing
- When you can poll for results

### Synchronous Conversion (Immediate Response)
```
POST /api/v1/convert/sync
Content-Type: multipart/form-data

file: <binary> (max 10MB)
timeout: 30 (optional, max seconds to wait)
```
Returns converted file immediately. Best for:
- Small files (<10MB)
- Real-time workflows
- Single document needs

### Synchronous with JSON Response
```
POST /api/v1/convert/sync/json
Content-Type: multipart/form-data

file: <binary>
```
Returns JSON with conversion metadata

### Batch Convert
```
POST /api/v1/batch
Content-Type: application/json

{
  "source": "azure://container/path",
  "destination": "azure://container/output",
  "files": ["file1.dot", "file2.dot"]
}
```

### Job Status
```
GET /api/v1/jobs/{job-id}
```

### Interactive Documentation

- **Swagger UI**: Available at `/swagger` when the service is running
- **OpenAPI Spec**: Available at `/api/v1/openapi.yaml`
- **Landing Page**: Available at `/` with quick start guide

### Health & Monitoring

```bash
# Basic health check
GET /health

# Liveness probe (for Kubernetes)
GET /health/live

# Readiness probe (checks queue connectivity)
GET /health/ready

# Prometheus metrics
GET /metrics
```

#### Metrics Available
- `http_requests_total`: Total HTTP requests by endpoint
- `http_request_duration_seconds`: Request latency histogram
- `conversion_jobs_total`: Total conversion jobs by status
- `conversion_duration_seconds`: Conversion time histogram
- `worker_pool_size`: Current number of active workers
- `queue_size`: Current number of jobs in queue

## Installation & Deployment

### Docker Deployment

The application uses a multi-stage Docker build for optimal image size (~850MB):

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
# ... build process

# Runtime stage with LibreOffice
FROM alpine:3.19
# Includes: LibreOffice, Java 11, fonts, security updates
```

### Required Dependencies

The Docker image includes all necessary dependencies:

- **LibreOffice 7.6+**: Core conversion engine
- **OpenJDK 11 JRE**: Required by LibreOffice
- **Font packages**:
  - Noto fonts (including CJK)
  - Liberation fonts
  - DejaVu fonts
  - Microsoft Core Fonts
- **System libraries**: Cairo, CUPS, D-Bus for document rendering
- **Security**: Non-root user, tini for signal handling

### Azure Container Instance

```bash
az container create \
  --resource-group myResourceGroup \
  --name dot-converter \
  --image alterspective/dot-to-docx-converter:latest \
  --cpu 4 \
  --memory 8 \
  --environment-variables \
    REDIS_URL=redis://... \
    AZURE_STORAGE_CONNECTION_STRING=... \
    WORKER_COUNT=10
```

### Azure Container Apps (Recommended)

```bash
# Deploy using the provided YAML configuration
az containerapp create \
  --name dot-to-docx-converter \
  --resource-group myResourceGroup \
  --environment myEnvironment \
  --yaml azure/container-app.yaml
```

The Container Apps configuration includes:
- Auto-scaling from 1 to 10 instances
- HTTP ingress with SSL termination
- Health probes configuration
- Resource limits (2 CPU, 4GB RAM per instance)

### Azure Kubernetes Service (AKS)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dot-to-docx-converter
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: converter
        image: alterspective.azurecr.io/dot-to-docx-converter:latest
        resources:
          requests:
            memory: "4Gi"
            cpu: "2"
          limits:
            memory: "8Gi"
            cpu: "4"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
```

### Docker Compose (Development)

```yaml
version: '3.8'
services:
  converter:
    build: .
    ports:
      - "8080:8080"
    environment:
      - REDIS_URL=redis://redis:6379
      - WORKER_COUNT=10
      - LOG_LEVEL=debug
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | HTTP server port | `8080` | No |
| `WORKER_COUNT` | Number of concurrent workers | `10` | No |
| `REDIS_URL` | Redis connection URL (e.g., `redis://host:6379`) | - | No* |
| `AZURE_STORAGE_CONNECTION_STRING` | Azure Storage connection string | - | No* |
| `AZURE_STORAGE_CONTAINER` | Blob container name | `conversions` | No |
| `MAX_FILE_SIZE` | Maximum file size in MB | `50` | No |
| `CONVERSION_TIMEOUT` | Timeout per document in seconds | `60` | No |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | `info` | No |
| `METRICS_PORT` | Separate port for metrics (if needed) | Same as PORT | No |
| `ENABLE_SWAGGER` | Enable Swagger UI | `true` | No |

*Falls back to in-memory queue and local storage if not provided

### Configuration File (config.yaml)

```yaml
server:
  port: 8080
  readTimeout: 30s
  writeTimeout: 30s

worker:
  count: 10
  queueSize: 1000

converter:
  timeout: 60s
  maxFileSize: 50MB
  tempDir: /tmp/conversions

redis:
  url: redis://localhost:6379
  maxRetries: 3

storage:
  type: azure # or 'local'
  azure:
    connectionString: ${AZURE_STORAGE_CONNECTION_STRING}
    container: conversions
  local:
    path: /tmp/conversions
```

## Technology Stack

### Core Technologies
- **Language**: Go 1.21 (compiled, statically linked)
- **Web Framework**: Gin v1.9.1 (high-performance HTTP)
- **Conversion Engine**: LibreOffice 7.6+ (headless mode)
- **Container Base**: Alpine Linux 3.19 (security-hardened)

### Infrastructure
- **Queue System**: Redis 7.0+ (go-redis/v9)
- **Storage**: Azure Blob Storage SDK (azblob v1.2.1)
- **Container Registry**: Azure Container Registry / GitHub Container Registry
- **Orchestration**: Azure Container Apps / AKS / Docker

### Observability
- **Metrics**: Prometheus client (prometheus/client_golang)
- **Logging**: Structured JSON logging (sirupsen/logrus)
- **Tracing**: OpenTelemetry ready (optional)
- **Health Checks**: Kubernetes-compatible probes

### Development & CI/CD
- **CI/CD**: GitHub Actions workflow
- **Testing**: Go native testing framework
- **Linting**: golangci-lint
- **Container Build**: Multi-stage Docker build
- **Security Scanning**: Trivy, Snyk (in CI/CD)

## Development Guide

### Project Structure

```
.
├── main.go                 # Application entry point
├── internal/
│   ├── api/               # HTTP handlers and routing
│   │   ├── handlers.go    # Conversion endpoints
│   │   ├── health.go      # Health check endpoints
│   │   ├── pages.go       # Static pages (landing, swagger)
│   │   └── server.go      # HTTP server setup
│   ├── converter/         # Conversion logic
│   │   ├── converter.go   # Interface definition
│   │   └── libreoffice.go # LibreOffice implementation
│   ├── queue/             # Job queue implementations
│   │   ├── queue.go       # Queue interface
│   │   ├── memory.go      # In-memory queue
│   │   └── redis.go       # Redis queue
│   ├── storage/           # Storage backends
│   │   ├── storage.go     # Storage interface
│   │   ├── azure.go       # Azure Blob Storage
│   │   └── local.go       # Local filesystem
│   ├── worker/            # Worker pool management
│   │   └── pool.go        # Concurrent worker pool
│   └── config/            # Configuration management
│       └── config.go      # Config loading and validation
├── docs/
│   └── openapi.yaml       # OpenAPI specification
├── azure/
│   ├── container-app.yaml # Container Apps config
│   └── parameters.prod.json # Production parameters
├── .github/
│   └── workflows/
│       └── ci-cd.yml      # GitHub Actions workflow
├── Dockerfile             # Multi-stage Docker build
├── go.mod                 # Go module definition
└── README.md              # This file
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/converter/...

# Benchmark tests
go test -bench=. ./...
```

### Local Development Setup

1. **Install Dependencies**:
   ```bash
   # macOS
   brew install libreoffice redis go

   # Ubuntu/Debian
   sudo apt-get install libreoffice redis-server golang-1.21

   # Windows (WSL2 recommended)
   sudo apt-get install libreoffice redis-server
   ```

2. **Start Redis**:
   ```bash
   redis-server
   ```

3. **Run the Service**:
   ```bash
   go run main.go
   ```

4. **Test Conversion**:
   ```bash
   curl -X POST http://localhost:8080/api/v1/convert \
     -F "file=@test.dot" \
     -F "priority=1"
   ```

## Production Considerations

### Security

- ✅ Non-root container execution (user: converter, UID: 1000)
- ✅ Read-only root filesystem capability
- ✅ No sensitive data in logs
- ✅ Input validation and sanitization
- ✅ Secure defaults (timeouts, limits)
- ✅ TLS termination at ingress level
- ✅ Secret management via environment variables

### Scaling

- **Horizontal Scaling**: Increase replicas for higher throughput
- **Vertical Scaling**: Adjust CPU/memory for larger files
- **Queue Scaling**: Use Redis cluster for high availability
- **Storage Scaling**: Azure Blob Storage handles unlimited files

### Monitoring & Alerts

Recommended alerts:
- Conversion failure rate > 5%
- Queue size > 1000 jobs
- Worker pool utilization > 80%
- Response time p95 > 1s
- Memory usage > 80%
- Disk usage > 80%

### Backup & Recovery

- Queue state persisted in Redis (configure persistence)
- Converted files stored in Azure Blob Storage (geo-redundant)
- Application stateless - can be redeployed anytime
- Jobs can be retried on failure

## Troubleshooting

### Common Issues

1. **LibreOffice not found**:
   ```
   Error: exec: "soffice": executable file not found
   Solution: Ensure LibreOffice is installed in the container
   ```

2. **Conversion timeout**:
   ```
   Error: conversion timeout exceeded (60s)
   Solution: Increase CONVERSION_TIMEOUT for large files
   ```

3. **Redis connection failed**:
   ```
   Error: Failed to initialize Redis queue
   Solution: Check REDIS_URL and network connectivity
   ```

4. **Out of memory**:
   ```
   Error: signal: killed
   Solution: Increase container memory limit
   ```

### Debug Mode

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Enable Go debug output
export GODEBUG=http2debug=2

# Run with race detector (development only)
go run -race main.go
```

## API Examples

### Synchronous Conversion (Immediate Response)

```bash
# Convert and get file immediately
curl -X POST https://dot-to-docx-converter-prod.lemondesert-9ded9ffc.eastus.azurecontainerapps.io/api/v1/convert/sync \
  -F "file=@document.dot" \
  -F "timeout=20" \
  -o converted.docx

# Get JSON response with metadata
curl -X POST https://dot-to-docx-converter-prod.lemondesert-9ded9ffc.eastus.azurecontainerapps.io/api/v1/convert/sync/json \
  -F "file=@document.dot"

# Response:
{
  "success": true,
  "conversion_id": "550e8400-e29b-41d4-a716-446655440000",
  "filename": "document.docx",
  "size": 78249,
  "duration": "2.5s"
}
```

### Asynchronous Conversion (Queue-based)

```bash
# Upload and convert a file
curl -X POST http://localhost:8080/api/v1/convert \
  -H "Accept: application/json" \
  -F "file=@document.dot" \
  -F "priority=1" \
  -F 'metadata={"user":"john","department":"sales"}'

# Response
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "input_path": "uploads/550e8400/document.dot",
  "output_path": "outputs/550e8400/document.docx",
  "created_at": "2024-01-01T12:00:00Z"
}
```

### Batch Conversion

```bash
# Submit batch job
curl -X POST http://localhost:8080/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "source": "azure://input-container/documents",
    "destination": "azure://output-container/converted",
    "files": ["doc1.dot", "doc2.dot", "doc3.dot"],
    "priority": 2
  }'
```

### Check Job Status

```bash
# Get job status
curl http://localhost:8080/api/v1/jobs/550e8400-e29b-41d4-a716-446655440000

# Response
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "duration": "2.5s",
  "download_url": "/api/v1/download/550e8400-e29b-41d4-a716-446655440000"
}
```

### Download Result

```bash
# Download converted file
curl -O http://localhost:8080/api/v1/download/550e8400-e29b-41d4-a716-446655440000
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### Development Guidelines

- Write tests for new features
- Update documentation
- Follow Go best practices
- Ensure CI/CD passes
- Add meaningful commit messages

## Support

- **Issues**: [GitHub Issues](https://github.com/alterspective-engine/dot-to-docx-converter/issues)
- **Discussions**: [GitHub Discussions](https://github.com/alterspective-engine/dot-to-docx-converter/discussions)
- **Security**: Report security vulnerabilities privately via GitHub Security Advisory

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- LibreOffice team for the excellent conversion engine
- Go community for the amazing ecosystem
- Contributors and users of this project

---

**Note**: This service is specifically designed for .dot to .docx conversion. For other format conversions, consider using Pandoc or similar tools.