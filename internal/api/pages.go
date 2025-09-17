package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ServeLandingPage serves the main landing page
func ServeLandingPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DOT to DOCX Converter Service</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }

        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            max-width: 900px;
            width: 100%%;
            padding: 50px;
            animation: slideUp 0.5s ease-out;
        }

        @keyframes slideUp {
            from {
                opacity: 0;
                transform: translateY(30px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        h1 {
            color: #667eea;
            font-size: 2.5rem;
            margin-bottom: 10px;
            text-align: center;
        }

        .subtitle {
            text-align: center;
            color: #666;
            font-size: 1.2rem;
            margin-bottom: 40px;
        }

        .section {
            margin-bottom: 40px;
        }

        h2 {
            color: #764ba2;
            font-size: 1.8rem;
            margin-bottom: 20px;
            border-bottom: 2px solid #f0f0f0;
            padding-bottom: 10px;
        }

        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-top: 20px;
        }

        .feature {
            padding: 20px;
            background: #f8f9fa;
            border-radius: 10px;
            border-left: 4px solid #667eea;
        }

        .feature h3 {
            color: #667eea;
            margin-bottom: 10px;
        }

        .endpoints {
            background: #f8f9fa;
            border-radius: 10px;
            padding: 20px;
            margin-top: 20px;
        }

        .endpoint {
            display: flex;
            align-items: center;
            margin-bottom: 15px;
            padding: 10px;
            background: white;
            border-radius: 5px;
        }

        .method {
            font-weight: bold;
            padding: 5px 10px;
            border-radius: 3px;
            margin-right: 15px;
            text-align: center;
            min-width: 60px;
        }

        .method.post {
            background: #49cc90;
            color: white;
        }

        .method.get {
            background: #61affe;
            color: white;
        }

        .path {
            font-family: 'Courier New', monospace;
            flex: 1;
        }

        .description {
            color: #666;
            font-size: 0.9rem;
            margin-left: 15px;
        }

        .buttons {
            display: flex;
            gap: 20px;
            justify-content: center;
            margin-top: 40px;
        }

        .button {
            display: inline-block;
            padding: 15px 30px;
            border-radius: 50px;
            text-decoration: none;
            font-weight: 600;
            transition: transform 0.2s, box-shadow 0.2s;
        }

        .button:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(0, 0, 0, 0.2);
        }

        .button.primary {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
        }

        .button.secondary {
            background: white;
            color: #667eea;
            border: 2px solid #667eea;
        }

        .status {
            display: flex;
            align-items: center;
            justify-content: center;
            margin-bottom: 30px;
        }

        .status-indicator {
            width: 12px;
            height: 12px;
            background: #49cc90;
            border-radius: 50%%;
            margin-right: 10px;
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0%%, 100%% {
                opacity: 1;
            }
            50%% {
                opacity: 0.5;
            }
        }

        .code-block {
            background: #2d2d2d;
            color: #f8f8f2;
            padding: 20px;
            border-radius: 10px;
            overflow-x: auto;
            margin-top: 20px;
        }

        .code-block code {
            font-family: 'Courier New', monospace;
            font-size: 14px;
        }

        @media (max-width: 768px) {
            .container {
                padding: 30px;
            }

            h1 {
                font-size: 2rem;
            }

            .buttons {
                flex-direction: column;
            }

            .button {
                width: 100%%;
                text-align: center;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="status">
            <span class="status-indicator"></span>
            <span>Service is operational</span>
        </div>

        <h1>DOT to DOCX Converter</h1>
        <p class="subtitle">High-performance document conversion service</p>

        <div class="section">
            <h2>About</h2>
            <p>
                This service provides enterprise-grade conversion of legacy Word template files (.dot) to modern Word format (.docx).
                Built with Go and LibreOffice, it offers reliable, scalable batch processing with asynchronous job management.
            </p>
        </div>

        <div class="section">
            <h2>Features</h2>
            <div class="features">
                <div class="feature">
                    <h3>🚀 High Performance</h3>
                    <p>Concurrent processing with worker pools and Redis-backed queue management</p>
                </div>
                <div class="feature">
                    <h3>📦 Batch Processing</h3>
                    <p>Convert single files or entire directories with up to 1000 files per batch</p>
                </div>
                <div class="feature">
                    <h3>☁️ Cloud Native</h3>
                    <p>Deployed on Azure Container Apps with auto-scaling and high availability</p>
                </div>
                <div class="feature">
                    <h3>📊 Monitoring</h3>
                    <p>Prometheus metrics and Application Insights integration for observability</p>
                </div>
            </div>
        </div>

        <div class="section">
            <h2>API Endpoints</h2>
            <div class="endpoints">
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/convert</span>
                    <span class="description">Convert single DOT file</span>
                </div>
                <div class="endpoint">
                    <span class="method post">POST</span>
                    <span class="path">/api/v1/batch</span>
                    <span class="description">Batch convert multiple files</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/jobs/{id}</span>
                    <span class="description">Check job status</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/api/v1/download/{id}</span>
                    <span class="description">Download converted file</span>
                </div>
                <div class="endpoint">
                    <span class="method get">GET</span>
                    <span class="path">/health</span>
                    <span class="description">Health check endpoint</span>
                </div>
            </div>
        </div>

        <div class="section">
            <h2>Quick Start</h2>
            <p>Convert a DOT file using curl:</p>
            <div class="code-block">
                <code>curl -X POST \<br>
  -F "file=@document.dot" \<br>
  %s/api/v1/convert</code>
            </div>
        </div>

        <div class="buttons">
            <a href="/swagger" class="button primary">View API Documentation</a>
            <a href="https://github.com/Alterspective-Engine/dot-to-docx-converter" class="button secondary">View on GitHub</a>
        </div>
    </div>
</body>
</html>`, c.Request.Host)

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}

// ServeSwaggerUI serves the Swagger UI page
func ServeSwaggerUI() gin.HandlerFunc {
	return func(c *gin.Context) {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DOT to DOCX Converter - API Documentation</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/swagger-ui.css">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
        .topbar {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 15px 30px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .topbar h1 {
            margin: 0;
            font-size: 1.5rem;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }
        .topbar a {
            color: white;
            text-decoration: none;
            padding: 8px 16px;
            border: 2px solid white;
            border-radius: 20px;
            transition: all 0.3s;
        }
        .topbar a:hover {
            background: white;
            color: #667eea;
        }
        #swagger-ui {
            margin-top: 0;
        }
        .swagger-ui .topbar {
            display: none;
        }
    </style>
</head>
<body>
    <div class="topbar">
        <h1>DOT to DOCX Converter API</h1>
        <a href="/">Back to Home</a>
    </div>
    <div id="swagger-ui"></div>

    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "/api/v1/openapi.yaml",
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                docExpansion: "list",
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 1,
                tryItOutEnabled: true
            });

            window.ui = ui;
        }
    </script>
</body>
</html>`

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}

// ServeOpenAPISpec serves the OpenAPI YAML specification
func ServeOpenAPISpec() gin.HandlerFunc {
	return func(c *gin.Context) {
		spec := `openapi: 3.0.3
info:
  title: DOT to DOCX Converter API
  description: |
    High-performance document conversion service for converting legacy Word template files (.dot) to modern Word format (.docx).

    This service provides a robust, scalable solution for batch converting DOT files using LibreOffice as the conversion engine.

    ## Features
    - Single file and batch conversion support
    - Asynchronous processing with job queue
    - Redis-backed job management
    - Auto-scaling based on load
    - Health monitoring endpoints
    - Prometheus metrics

    ## Usage
    1. Upload a .dot file using POST /api/v1/convert
    2. Monitor job status using GET /api/v1/jobs/{id}
    3. Download the converted file when complete

  version: 1.0.0
  contact:
    name: Alterspective Engine
    url: https://github.com/Alterspective-Engine/dot-to-docx-converter
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT

servers:
  - url: /
    description: Current server

tags:
  - name: Conversion
    description: Document conversion operations
  - name: Jobs
    description: Job management operations
  - name: Health
    description: Health check endpoints
  - name: Metrics
    description: Monitoring endpoints

paths:
  /api/v1/convert:
    post:
      summary: Convert single DOT file to DOCX
      description: Upload a .dot file for conversion to .docx format. The conversion is processed asynchronously.
      tags:
        - Conversion
      requestBody:
        required: true
        content:
          multipart/form-data:
            schema:
              type: object
              required:
                - file
              properties:
                file:
                  type: string
                  format: binary
                  description: The .dot file to convert
                priority:
                  type: integer
                  default: 0
                  description: Job priority (higher values processed first)
                metadata:
                  type: object
                  additionalProperties:
                    type: string
                  description: Optional metadata to attach to the job
      responses:
        '202':
          description: Job accepted for processing
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/JobResponse'
        '400':
          description: Invalid request (file missing or wrong format)
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
        '500':
          description: Internal server error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

  /api/v1/batch:
    post:
      summary: Batch convert multiple DOT files
      description: Submit multiple files for batch conversion. Useful for processing entire directories.
      tags:
        - Conversion
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/BatchConvertRequest'
      responses:
        '202':
          description: Batch job accepted
          content:
            application/json:
              schema:
                type: object
                properties:
                  batch_id:
                    type: string
                    format: uuid
                    description: Unique batch identifier
                  jobs:
                    type: array
                    items:
                      $ref: '#/components/schemas/JobResponse'
                  count:
                    type: integer
                    description: Number of jobs created
        '400':
          description: Invalid request
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

  /api/v1/jobs/{id}:
    get:
      summary: Get job status
      description: Retrieve the current status and details of a conversion job
      tags:
        - Jobs
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
          description: Job ID
      responses:
        '200':
          description: Job details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/JobResponse'
        '404':
          description: Job not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

  /api/v1/jobs/{id}/cancel:
    post:
      summary: Cancel a job
      description: Cancel a pending conversion job
      tags:
        - Jobs
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
          description: Job ID
      responses:
        '200':
          description: Job cancelled
          content:
            application/json:
              schema:
                type: object
                properties:
                  message:
                    type: string
        '404':
          description: Job not found
        '400':
          description: Job cannot be cancelled (already processing or completed)

  /api/v1/jobs:
    get:
      summary: List jobs
      description: List all jobs with optional status filtering
      tags:
        - Jobs
      parameters:
        - name: status
          in: query
          schema:
            type: string
            enum: [pending, processing, completed, failed]
          description: Filter by job status
      responses:
        '200':
          description: Job list
          content:
            application/json:
              schema:
                type: object
                properties:
                  jobs:
                    type: array
                    items:
                      $ref: '#/components/schemas/JobResponse'
                  count:
                    type: integer

  /api/v1/download/{id}:
    get:
      summary: Download converted file
      description: Download the converted DOCX file for a completed job
      tags:
        - Jobs
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
          description: Job ID
      responses:
        '200':
          description: Converted file
          content:
            application/vnd.openxmlformats-officedocument.wordprocessingml.document:
              schema:
                type: string
                format: binary
        '404':
          description: File not found or job not completed

  /health:
    get:
      summary: Health check
      description: Basic health check endpoint
      tags:
        - Health
      responses:
        '200':
          description: Service is healthy
          content:
            application/json:
              schema:
                type: object
                properties:
                  service:
                    type: string
                  status:
                    type: string
                  version:
                    type: string

  /health/ready:
    get:
      summary: Readiness check
      description: Indicates if the service is ready to accept requests
      tags:
        - Health
      responses:
        '200':
          description: Service is ready
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string

  /health/live:
    get:
      summary: Liveness check
      description: Indicates if the service is alive
      tags:
        - Health
      responses:
        '200':
          description: Service is alive
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string

  /metrics:
    get:
      summary: Prometheus metrics
      description: Metrics endpoint for Prometheus scraping
      tags:
        - Metrics
      responses:
        '200':
          description: Prometheus metrics
          content:
            text/plain:
              schema:
                type: string

components:
  schemas:
    JobResponse:
      type: object
      properties:
        job_id:
          type: string
          format: uuid
          description: Unique job identifier
        status:
          type: string
          enum: [pending, processing, completed, failed]
          description: Current job status
        input_path:
          type: string
          description: Path to the input file
        output_path:
          type: string
          description: Path to the output file
        created_at:
          type: string
          format: date-time
          description: Job creation timestamp
        started_at:
          type: string
          format: date-time
          description: Processing start time
        completed_at:
          type: string
          format: date-time
          description: Processing completion time
        duration:
          type: string
          description: Processing duration
        error:
          type: string
          description: Error message if job failed
        download_url:
          type: string
          description: URL to download the converted file

    BatchConvertRequest:
      type: object
      required:
        - source
        - destination
        - files
      properties:
        source:
          type: string
          description: Source directory path
        destination:
          type: string
          description: Destination directory path
        files:
          type: array
          items:
            type: string
          description: List of file paths to convert
          maxItems: 1000
        priority:
          type: integer
          default: 0
          description: Job priority

    Error:
      type: object
      properties:
        error:
          type: string
          description: Error message`

		c.Data(http.StatusOK, "application/x-yaml; charset=utf-8", []byte(spec))
	}
}