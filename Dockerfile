FROM golang:1.24-alpine AS builder


WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o janus-services \
    cmd/processor/main.go

# Create minimal production image
FROM alpine:latest

# Create non-root user
RUN adduser -D -u 1000 appuser

# Install only necessary runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/janus-services .

# Create necessary directories
RUN mkdir -p /recordings \
    && mkdir -p /var/log/janus-processor \
    && mkdir -p /opt/janus/share/janus/recordings \
    && chown -R appuser:appuser /app /recordings /var/log/janus-processor /opt/janus/share/janus/recordings

# Set environment variable for janus-pp-rec path
ENV JANUS_PP_REC_PATH=/usr/local/bin/janus-pp-rec

# Switch to non-root user
USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/janus-services"]