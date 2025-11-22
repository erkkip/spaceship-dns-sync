# Build stage
FROM golang:1.22.4-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dnsupdater ./cmd/dnsupdater

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 dnsupdater && \
    adduser -D -u 1000 -G dnsupdater dnsupdater

WORKDIR /app

# Create state directory with proper permissions
RUN mkdir -p /app/state && \
    chown -R dnsupdater:dnsupdater /app

# Copy binary from builder
COPY --from=builder /build/dnsupdater /usr/local/bin/dnsupdater

# Switch to non-root user
USER dnsupdater

# Set default cache path to /app/state/last_ip
ENV CACHE_PATH=/app/state/last_ip

ENTRYPOINT ["/usr/local/bin/dnsupdater"]

