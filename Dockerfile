FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .

RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o bolt-loadbalancer \
    ./cmd

RUN ./bolt-loadbalancer --version || echo "Binary built successfully"

FROM alpine:3.18

RUN apk --no-cache add \
    ca-certificates \
    curl \
    tzdata \
    && rm -rf /var/cache/apk/*

RUN addgroup -g 1001 -S bolt && \
    adduser -u 1001 -S bolt -G bolt -h /app

WORKDIR /app

COPY --from=builder /app/bolt-loadbalancer .
COPY --from=builder /app/sample_config.yaml ./sample_config.yaml

RUN mkdir -p /app/config && \
    chown -R bolt:bolt /app

USER bolt

EXPOSE 8100
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8100/health || exit 1

CMD ["./bolt-loadbalancer", "-c", "config.yaml"]