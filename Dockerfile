# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with version info
ARG VERSION=dev
ARG COMMIT=local
ARG BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.BuildVersion=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.BuildCommit=${COMMIT}" \
    -o financeapi ./main.go

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Set timezone
ENV TZ=Asia/Jakarta

# Create non-root user
RUN addgroup -g 1000 appgroup && adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/financeapi .
COPY --from=builder /app/.env.example .env
COPY --from=builder /app/public ./public
COPY --from=builder /app/essentials/knowledge ./essentials/knowledge
# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port 8000
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8000/health/build || exit 1

# Run the application
CMD ["./financeapi"]
