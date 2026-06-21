# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /build/bin/p2p-gateway ./cmd/p2p-gateway/

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata python3 ffmpeg \
    && addgroup -g 1000 app \
    && adduser -D -u 1000 -G app app

WORKDIR /app

# Copy binary
COPY --from=builder /build/bin/p2p-gateway /app/p2p-gateway

# Copy config
COPY config.yaml /app/config.yaml

# Copy helper binaries
COPY bin/ /app/bin/

# Security: non-root
USER app:app

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:8082/health || exit 1

EXPOSE 8082 8554

ENTRYPOINT ["/app/p2p-gateway"]