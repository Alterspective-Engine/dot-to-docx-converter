# DOT to DOCX Converter Service

High-performance document conversion service for converting legacy Word template files (.dot) to modern Word format (.docx). Optimized for Azure Container Instances with support for processing 500-900 documents efficiently.

## Features

- **High Performance**: Concurrent processing with worker pools
- **Azure Optimized**: Designed for Azure Container Instances/AKS
- **REST API**: Simple HTTP interface for conversion requests
- **Batch Processing**: Support for bulk document conversion
- **Queue System**: Redis-based job queue for reliability
- **Monitoring**: Prometheus metrics and health checks
- **Storage Integration**: Azure Blob Storage support
- **Accuracy**: LibreOffice-based conversion for maximum compatibility

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   REST API  │────▶│  Job Queue   │────▶│   Workers   │
└─────────────┘     │   (Redis)    │     │  (Pool of   │
                    └──────────────┘     │  Converters)│
                                         └─────────────┘
                                                │
                    ┌──────────────┐           ▼
                    │ Azure Blob   │◀─────────────────
                    │   Storage     │
                    └──────────────┘
```

## Performance Targets

- **Throughput**: 500-900 documents in 2-3 hours
- **Concurrency**: 10-20 parallel conversions
- **Memory**: Optimized for 4-8GB containers
- **CPU**: Efficient with 2-4 vCPUs

## API Endpoints

### Convert Single Document
```
POST /api/v1/convert
Content-Type: multipart/form-data

file: <binary>
```

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

### Health Check
```
GET /health
GET /metrics  # Prometheus format
```

## Deployment

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

### Docker

```bash
docker build -t dot-to-docx-converter .
docker run -p 8080:8080 \
  -e REDIS_URL=redis://localhost:6379 \
  -e WORKER_COUNT=10 \
  dot-to-docx-converter
```

## Configuration

Environment variables:

- `PORT`: HTTP port (default: 8080)
- `WORKER_COUNT`: Number of concurrent workers (default: 10)
- `REDIS_URL`: Redis connection URL
- `AZURE_STORAGE_CONNECTION_STRING`: Azure Storage connection
- `MAX_FILE_SIZE`: Maximum file size in MB (default: 50)
- `CONVERSION_TIMEOUT`: Timeout per document in seconds (default: 60)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

## Technology Stack

- **Language**: Go 1.21
- **Conversion Engine**: LibreOffice 7.6
- **Queue**: Redis 7
- **Storage**: Azure Blob Storage
- **Monitoring**: Prometheus
- **Container**: Alpine Linux

## License

MIT