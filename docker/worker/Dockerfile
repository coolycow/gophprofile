FROM golang:1.26.1-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG GO_WORKER_MAIN_PATH=./cmd/worker
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/worker "${GO_WORKER_MAIN_PATH}"

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 65532 appuser

COPY --from=builder /out/worker /usr/local/bin/worker

WORKDIR /app

USER appuser

EXPOSE 8000

# Настройки (DATABASE_DSN, MINIO_*, и т.д.) задаются через environment в docker-compose или .env — см. .env.example
CMD ["worker"]
