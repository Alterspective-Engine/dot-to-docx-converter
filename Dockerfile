# Multi-stage build for optimal size and security
# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Copy source code first
COPY . .

# Download and verify dependencies
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o converter \
    main.go

# Stage 2: Final image with LibreOffice
FROM alpine:3.19

# Install LibreOffice and required dependencies
RUN apk add --no-cache \
    libreoffice \
    openjdk11-jre-headless \
    font-noto \
    font-noto-cjk \
    font-noto-extra \
    ttf-liberation \
    ttf-dejavu \
    msttcorefonts-installer \
    fontconfig \
    dbus-x11 \
    cairo \
    cups-libs \
    libsm \
    libxt \
    && fc-cache -f \
    && update-ms-fonts \
    && fc-cache -f

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    tini

# Create non-root user
RUN addgroup -g 1000 converter && \
    adduser -D -u 1000 -G converter converter

# Create necessary directories
RUN mkdir -p /tmp/conversions /app/logs /home/converter/.config \
    && chown -R converter:converter /tmp/conversions /app/logs /home/converter

# Copy binary from builder
COPY --from=builder /app/converter /app/converter

# Copy static files and documentation
COPY --chown=converter:converter ./static /app/static
COPY --chown=converter:converter ./docs /app/docs

# Set working directory
WORKDIR /app

# Switch to non-root user
USER converter

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Expose port
EXPOSE 8080

# Use tini as entrypoint for proper signal handling
ENTRYPOINT ["/sbin/tini", "--"]

# Run the application
CMD ["/app/converter"]