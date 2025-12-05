# Build stage
FROM golang:1.24.3-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o monitrix ./cmd/monitrix

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/monitrix .

# Copy web assets
COPY web/ ./web/

# Create data directory
RUN mkdir -p /app/data

# Expose web server port
EXPOSE 8080

# Set default environment variables
ENV MONITOR_HOSTS="1.1.1.1,8.8.8.8,google.com,cloudflare.com,github.com"
ENV MONITOR_INTERVAL="30"
ENV WEB_ADDR="0.0.0.0:8080"

# Run the application
CMD ["./monitrix"]
