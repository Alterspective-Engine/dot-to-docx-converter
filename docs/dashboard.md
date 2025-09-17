# Metrics Dashboard

## Overview

The DOT to DOCX Converter service now includes a comprehensive metrics dashboard that provides real-time monitoring and visualization of document processing metrics.

## Features

### Real-Time Metrics
- **Processing Now**: Current number of documents being processed
- **Total Processed**: Cumulative count of successfully converted documents
- **Success Rate**: Percentage of successful conversions
- **Average Processing Time**: Mean time taken for conversions

### Visual Charts
1. **Timeline Chart**: 24-hour history of documents processed over time
2. **Queue Status Chart**: Distribution of jobs (pending, processing, completed, failed)
3. **Performance Chart**: Response time distribution histogram

### System Resources
- CPU Usage monitoring
- Memory Usage tracking
- Worker Pool utilization
- Queue Size indicator

## Access

The dashboard is available at:
- **Local**: http://localhost:8080/dashboard
- **Production**: https://your-domain/dashboard

## API Endpoints

### Metrics API
- **Endpoint**: `/api/v1/metrics`
- **Method**: GET
- **Response**: JSON with current metrics data

Example response:
```json
{
  "processing": 5,
  "total_processed": 1250,
  "total_failed": 3,
  "success_rate": 0.998,
  "avg_processing_time": 2.3,
  "timeline": [...],
  "queue_status": {
    "pending": 12,
    "processing": 5,
    "completed": 1250,
    "failed": 3
  },
  "system": {
    "cpu_usage": 45.2,
    "memory_usage": 2254438400,
    "memory_usage_percent": 60.5,
    "workers_active": 5,
    "workers_total": 10,
    "queue_size": 12
  }
}
```

## Branding

The dashboard follows Alterspective brand guidelines:
- **Primary Colors**: Navy (#17232D), Marine (#075156)
- **Secondary Colors**: Green (#2C8248), Citrus (#ABDD65)
- **Typography**: Montserrat font family
- **Logo**: Alterspective logo in header

## Technical Implementation

### Frontend
- Pure HTML/CSS/JavaScript
- Chart.js for data visualization
- Material Design 3 principles
- Responsive design for mobile compatibility

### Backend
- Go-based metrics collection
- Real-time data updates
- Integration with queue system
- System resource monitoring

## Auto-Refresh

The dashboard automatically refreshes metrics every 30 seconds. Manual refresh is available via the refresh button in the header.

## Future Enhancements

Planned improvements include:
- WebSocket support for real-time updates
- Historical data retention and analysis
- Custom date range selection
- Export metrics to CSV/JSON
- Alert configuration for thresholds
- Integration with monitoring platforms (Grafana, Datadog)