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
    <title>DOT to DOCX Converter - Alterspective</title>
    <link href="https://fonts.googleapis.com/css2?family=Montserrat:wght@400;500;600;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Montserrat', sans-serif;
            line-height: 1.6;
            color: #17232D;
            background: linear-gradient(135deg, #075156 0%%, #2C8248 50%%, #075156 100%%);
            min-height: 100vh;
            position: relative;
            overflow-x: hidden;
        }

        /* Animated background */
        body::before {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            width: 100%%;
            height: 100%%;
            background-image:
                radial-gradient(circle at 20%% 40%%, rgba(171, 221, 101, 0.1) 0%%, transparent 50%%),
                radial-gradient(circle at 80%% 60%%, rgba(44, 130, 72, 0.1) 0%%, transparent 50%%),
                radial-gradient(circle at 40%% 80%%, rgba(7, 81, 86, 0.1) 0%%, transparent 50%%);
            animation: backgroundShift 20s ease-in-out infinite;
            z-index: -1;
        }

        @keyframes backgroundShift {
            0%%, 100%% {
                transform: scale(1) rotate(0deg);
            }
            50%% {
                transform: scale(1.1) rotate(5deg);
            }
        }

        .hero-section {
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 40px 20px;
        }

        .container {
            background: rgba(255, 255, 255, 0.98);
            border-radius: 30px;
            box-shadow:
                0 30px 60px rgba(0, 0, 0, 0.2),
                0 0 100px rgba(171, 221, 101, 0.1);
            max-width: 1200px;
            width: 100%%;
            padding: 60px;
            animation: slideUp 0.8s ease-out;
            backdrop-filter: blur(10px);
        }

        @keyframes slideUp {
            from {
                opacity: 0;
                transform: translateY(50px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .header {
            text-align: center;
            margin-bottom: 60px;
        }

        .logo-container {
            display: inline-block;
            margin-bottom: 30px;
            position: relative;
            animation: logoReveal 1.2s ease-out forwards;
        }

        @keyframes logoReveal {
            0%% {
                opacity: 0;
                transform: scale(0.5) rotate(-180deg);
            }
            50%% {
                opacity: 1;
                transform: scale(1.1) rotate(10deg);
            }
            100%% {
                opacity: 1;
                transform: scale(1) rotate(0deg);
            }
        }

        .logo {
            height: 80px;
            filter: drop-shadow(0 10px 20px rgba(0, 0, 0, 0.2));
            position: relative;
        }

        .logo-container::before {
            content: '';
            position: absolute;
            top: 50%%;
            left: 50%%;
            width: 120%%;
            height: 120%%;
            background: radial-gradient(circle, rgba(171, 221, 101, 0.4) 0%%, transparent 70%%);
            transform: translate(-50%%, -50%%);
            animation: logoPulse 2s ease-out;
            opacity: 0;
            pointer-events: none;
        }

        @keyframes logoPulse {
            0%% {
                opacity: 0;
                transform: translate(-50%%, -50%%) scale(0.5);
            }
            50%% {
                opacity: 1;
            }
            100%% {
                opacity: 0;
                transform: translate(-50%%, -50%%) scale(2);
            }
        }

        h1 {
            color: #075156;
            font-size: 3rem;
            font-weight: 700;
            margin-bottom: 15px;
            background: linear-gradient(135deg, #075156 0%%, #2C8248 100%%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .tagline {
            color: #17232D;
            font-size: 1.3rem;
            font-weight: 500;
            margin-bottom: 20px;
        }

        .subtitle {
            color: #666;
            font-size: 1.1rem;
            margin-bottom: 30px;
        }

        .status-badge {
            display: inline-flex;
            align-items: center;
            gap: 10px;
            background: linear-gradient(135deg, #ABDD65 0%%, #2C8248 100%%);
            color: white;
            padding: 12px 24px;
            border-radius: 30px;
            font-weight: 600;
            animation: statusPulse 2s ease-in-out infinite;
        }

        @keyframes statusPulse {
            0%%, 100%% {
                transform: scale(1);
            }
            50%% {
                transform: scale(1.05);
            }
        }

        .status-indicator {
            width: 10px;
            height: 10px;
            background: white;
            border-radius: 50%%;
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0%%, 100%% {
                opacity: 1;
            }
            50%% {
                opacity: 0.3;
            }
        }

        .content-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 40px;
            margin-bottom: 50px;
        }

        .section {
            background: linear-gradient(135deg, #f8f9fa 0%%, #E5EEEF 100%%);
            border-radius: 20px;
            padding: 35px;
            position: relative;
            transition: all 0.3s ease;
            border: 2px solid transparent;
        }

        .section:hover {
            transform: translateY(-5px);
            box-shadow: 0 15px 40px rgba(7, 81, 86, 0.15);
            border-color: #ABDD65;
        }

        .section-icon {
            position: absolute;
            top: 35px;
            right: 35px;
            font-size: 2rem;
            color: #ABDD65;
            opacity: 0.5;
        }

        h2 {
            color: #075156;
            font-size: 1.8rem;
            margin-bottom: 20px;
            font-weight: 600;
        }

        .section p {
            color: #555;
            line-height: 1.8;
            margin-bottom: 20px;
        }

        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
            gap: 25px;
            margin-bottom: 50px;
        }

        .feature {
            padding: 30px;
            background: white;
            border-radius: 15px;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.08);
            transition: all 0.3s ease;
            border: 2px solid transparent;
            position: relative;
            overflow: hidden;
        }

        .feature::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 5px;
            height: 100%%;
            background: linear-gradient(180deg, #075156 0%%, #ABDD65 100%%);
        }

        .feature:hover {
            transform: translateY(-8px);
            box-shadow: 0 20px 40px rgba(7, 81, 86, 0.2);
            border-color: #ABDD65;
        }

        .feature-icon {
            font-size: 2.5rem;
            margin-bottom: 15px;
            background: linear-gradient(135deg, #075156 0%%, #2C8248 100%%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .feature h3 {
            color: #075156;
            font-size: 1.3rem;
            margin-bottom: 12px;
            font-weight: 600;
        }

        .feature p {
            color: #666;
            line-height: 1.7;
        }

        .api-section {
            background: linear-gradient(135deg, #075156 0%%, #2C8248 100%%);
            border-radius: 20px;
            padding: 40px;
            margin-bottom: 50px;
            position: relative;
            overflow: hidden;
        }

        .api-section::before {
            content: '';
            position: absolute;
            top: -50%%;
            right: -50%%;
            width: 200%%;
            height: 200%%;
            background: radial-gradient(circle, rgba(171, 221, 101, 0.1) 0%%, transparent 70%%);
            animation: rotate 30s linear infinite;
        }

        @keyframes rotate {
            from {
                transform: rotate(0deg);
            }
            to {
                transform: rotate(360deg);
            }
        }

        .api-content {
            position: relative;
            z-index: 1;
            text-align: center;
        }

        .api-section h2 {
            color: white;
            font-size: 2rem;
            margin-bottom: 20px;
        }

        .api-section p {
            color: rgba(255, 255, 255, 0.9);
            font-size: 1.1rem;
            margin-bottom: 30px;
            max-width: 600px;
            margin-left: auto;
            margin-right: auto;
        }

        .api-features {
            display: flex;
            justify-content: center;
            gap: 40px;
            margin-bottom: 30px;
            flex-wrap: wrap;
        }

        .api-feature {
            color: white;
            text-align: center;
        }

        .api-feature i {
            font-size: 2rem;
            margin-bottom: 10px;
            color: #ABDD65;
        }

        .api-feature span {
            display: block;
            font-weight: 600;
        }

        .code-preview {
            background: rgba(0, 0, 0, 0.3);
            border-radius: 15px;
            padding: 25px;
            margin-top: 30px;
            backdrop-filter: blur(10px);
            border: 1px solid rgba(171, 221, 101, 0.3);
        }

        .code-preview code {
            color: #ABDD65;
            font-family: 'Courier New', monospace;
            font-size: 14px;
            display: block;
            text-align: left;
        }

        .buttons {
            display: flex;
            gap: 20px;
            justify-content: center;
            flex-wrap: wrap;
        }

        .button {
            display: inline-flex;
            align-items: center;
            gap: 10px;
            padding: 18px 36px;
            border-radius: 50px;
            text-decoration: none;
            font-weight: 600;
            font-size: 1.1rem;
            transition: all 0.3s ease;
            position: relative;
            overflow: hidden;
        }

        .button::before {
            content: '';
            position: absolute;
            top: 0;
            left: -100%%;
            width: 100%%;
            height: 100%%;
            background: rgba(255, 255, 255, 0.1);
            transition: left 0.5s ease;
        }

        .button:hover::before {
            left: 100%%;
        }

        .button.primary {
            background: linear-gradient(135deg, #075156 0%%, #2C8248 100%%);
            color: white;
            box-shadow: 0 10px 30px rgba(7, 81, 86, 0.3);
        }

        .button.primary:hover {
            transform: translateY(-3px);
            box-shadow: 0 15px 40px rgba(7, 81, 86, 0.4);
        }

        .button.secondary {
            background: white;
            color: #075156;
            border: 2px solid #075156;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
        }

        .button.secondary:hover {
            background: #075156;
            color: white;
            transform: translateY(-3px);
            box-shadow: 0 15px 40px rgba(7, 81, 86, 0.3);
        }

        .button.accent {
            background: linear-gradient(135deg, #ABDD65 0%%, #2C8248 100%%);
            color: white;
            box-shadow: 0 10px 30px rgba(171, 221, 101, 0.3);
        }

        .button.accent:hover {
            transform: translateY(-3px);
            box-shadow: 0 15px 40px rgba(171, 221, 101, 0.4);
        }

        @media (max-width: 968px) {
            .content-grid {
                grid-template-columns: 1fr;
            }
        }

        @media (max-width: 768px) {
            .container {
                padding: 40px 30px;
            }

            h1 {
                font-size: 2.2rem;
            }

            .tagline {
                font-size: 1.1rem;
            }

            .buttons {
                flex-direction: column;
            }

            .button {
                width: 100%%;
                justify-content: center;
            }

            .api-features {
                flex-direction: column;
                gap: 20px;
            }
        }
    </style>
</head>
<body>
    <div class="hero-section">
        <div class="container">
            <div class="header">
                <div class="logo-container">
                    <img src="/static/alterspective-logo.png" alt="Alterspective" class="logo">
                </div>
                <h1>DOT to DOCX Converter</h1>
                <p class="tagline">Enterprise Document Conversion Service</p>
                <p class="subtitle">Powered by Alterspective Technology</p>
                <div class="status-badge">
                    <span class="status-indicator"></span>
                    <span>Service Operational</span>
                </div>
            </div>

            <div class="content-grid">
                <div class="section">
                    <i class="fas fa-info-circle section-icon"></i>
                    <h2>About the Service</h2>
                    <p>
                        Transform legacy Word template files (.dot) to modern Word format (.docx) with our enterprise-grade conversion service.
                    </p>
                    <p>
                        Built on cutting-edge Go architecture with LibreOffice integration, delivering reliable, scalable batch processing through asynchronous job management.
                    </p>
                </div>

                <div class="section">
                    <i class="fas fa-code section-icon"></i>
                    <h2>Quick Start</h2>
                    <p>Get started with a simple API call:</p>
                    <div class="code-preview" style="background: #2d2d2d; padding: 20px;">
                        <code style="color: #f8f8f2;">curl -X POST \<br>  -F "file=@document.dot" \<br>  %s/api/v1/convert</code>
                    </div>
                </div>
            </div>

            <div class="features">
                <div class="feature">
                    <div class="feature-icon">
                        <i class="fas fa-rocket"></i>
                    </div>
                    <h3>High Performance</h3>
                    <p>Concurrent processing with worker pools and Redis-backed queue management for optimal throughput</p>
                </div>
                <div class="feature">
                    <div class="feature-icon">
                        <i class="fas fa-layer-group"></i>
                    </div>
                    <h3>Batch Processing</h3>
                    <p>Convert single files or entire directories with support for up to 1000 files per batch operation</p>
                </div>
                <div class="feature">
                    <div class="feature-icon">
                        <i class="fas fa-cloud"></i>
                    </div>
                    <h3>Cloud Native</h3>
                    <p>Deployed on Azure Container Apps with auto-scaling, high availability, and global reach</p>
                </div>
                <div class="feature">
                    <div class="feature-icon">
                        <i class="fas fa-chart-line"></i>
                    </div>
                    <h3>Observability</h3>
                    <p>Prometheus metrics and Application Insights integration for complete system observability</p>
                </div>
            </div>

            <div class="api-section">
                <div class="api-content">
                    <h2>Comprehensive API Documentation</h2>
                    <p>
                        Explore our RESTful API with interactive documentation, code examples, and real-time testing capabilities
                    </p>
                    <div class="api-features">
                        <div class="api-feature">
                            <i class="fas fa-book"></i>
                            <span>OpenAPI 3.0</span>
                        </div>
                        <div class="api-feature">
                            <i class="fas fa-flask"></i>
                            <span>Try It Out</span>
                        </div>
                        <div class="api-feature">
                            <i class="fas fa-code"></i>
                            <span>Code Samples</span>
                        </div>
                        <div class="api-feature">
                            <i class="fas fa-shield-alt"></i>
                            <span>Authentication</span>
                        </div>
                    </div>
                    <div class="buttons">
                        <a href="/swagger" class="button accent">
                            <i class="fas fa-book-open"></i>
                            View API Documentation
                        </a>
                    </div>
                </div>
            </div>

            <div class="buttons">
                <a href="/dashboard" class="button primary">
                    <i class="fas fa-tachometer-alt"></i>
                    View Dashboard
                </a>
                <a href="https://github.com/Alterspective-Engine/dot-to-docx-converter" class="button secondary">
                    <i class="fab fa-github"></i>
                    View on GitHub
                </a>
            </div>
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
    <link href="https://fonts.googleapis.com/css2?family=Montserrat:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
        .topbar {
            background: linear-gradient(135deg, #075156 0%, #2C8248 100%);
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
            font-family: 'Montserrat', sans-serif;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 1rem;
        }
        .logo {
            height: 35px;
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
            color: #075156;
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
        <h1>
            <img src="/static/alterspective-logo.png" alt="Alterspective" class="logo">
            <span>DOT to DOCX Converter API</span>
        </h1>
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
      summary: Convert single DOT file to DOCX (Asynchronous)
      description: Upload a .dot file for conversion to .docx format. The conversion is processed asynchronously in a queue. Use this for large files or batch processing.
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

  /api/v1/convert/sync:
    post:
      summary: Convert single DOT file to DOCX (Synchronous)
      description: |
        Upload a .dot file for immediate conversion to .docx format. The conversion is processed synchronously and the converted file is returned directly in the response.

        **Limitations:**
        - Maximum file size: 10MB (configurable via SYNC_MAX_FILE_SIZE)
        - Maximum timeout: 30 seconds (configurable via SYNC_TIMEOUT)
        - Not suitable for batch processing

        **Use cases:**
        - Small files that need immediate conversion
        - Real-time conversion workflows
        - Single document processing where waiting is acceptable
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
                  description: The .dot file to convert (max 10MB)
                timeout:
                  type: integer
                  default: 30
                  minimum: 1
                  maximum: 60
                  description: Maximum time to wait for conversion (seconds)
      responses:
        '200':
          description: Conversion successful, returns the converted DOCX file
          content:
            application/vnd.openxmlformats-officedocument.wordprocessingml.document:
              schema:
                type: string
                format: binary
          headers:
            Content-Disposition:
              schema:
                type: string
                example: 'attachment; filename="document.docx"'
            X-Conversion-Time:
              schema:
                type: string
                example: "2.5s"
            X-Conversion-ID:
              schema:
                type: string
                format: uuid
                example: "550e8400-e29b-41d4-a716-446655440000"
        '400':
          description: Invalid request (file missing, wrong format, or too large)
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
        '408':
          description: Conversion timeout exceeded
          content:
            application/json:
              schema:
                type: object
                properties:
                  error:
                    type: string
                    example: "conversion timeout exceeded"
                  timeout:
                    type: number
                    example: 30
                  suggestion:
                    type: string
                    example: "use async endpoint /api/v1/convert for complex files"
        '500':
          description: Conversion failed
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

  /api/v1/convert/sync/json:
    post:
      summary: Convert DOT to DOCX with JSON response (Synchronous)
      description: |
        Same as /convert/sync but returns a JSON response with conversion metadata instead of the binary file.
        Useful for API clients that need structured responses.
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
                  description: The .dot file to convert (max 10MB)
      responses:
        '200':
          description: Conversion successful
          content:
            application/json:
              schema:
                type: object
                properties:
                  success:
                    type: boolean
                    example: true
                  conversion_id:
                    type: string
                    format: uuid
                    example: "550e8400-e29b-41d4-a716-446655440000"
                  filename:
                    type: string
                    example: "document.docx"
                  size:
                    type: integer
                    example: 78249
                    description: File size in bytes
                  duration:
                    type: string
                    example: "2.5s"
                  download_url:
                    type: string
                    example: "/api/v1/sync/download/550e8400-e29b-41d4-a716-446655440000"
        '400':
          description: Invalid request
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
        '408':
          description: Timeout exceeded
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