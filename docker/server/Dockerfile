FROM golang:1.26.1-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG GO_SERVER_MAIN_PATH=./cmd/server
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG BUILD_VERSION=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags="-s -w -X main.buildVersion=${BUILD_VERSION} -X main.buildDate=${BUILD_DATE} -X main.buildCommit=${GIT_COMMIT}" \
    -o /out/server "${GO_SERVER_MAIN_PATH}" && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags="-s -w" \
    -o /out/migrate ./cmd/migrate

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -H -u 65532 appuser

COPY --from=builder /out/server /usr/local/bin/server
COPY --from=builder /out/migrate /usr/local/bin/migrate

# Не в WORKDIR /app: том compose может монтироваться на /app и скрывать файлы образа.
COPY migrations /usr/local/share/gophprofile/migrations
ENV MIGRATIONS_PATH=/usr/local/share/gophprofile/migrations

# Том compose часто смонтирован на WORKDIR /app под root — запись audit.json в cwd падает с permission denied.
ENV AUDIT_FILE=/tmp/audit.json

# Слушать все интерфейсы; дефолт Go/конфига 127.0.0.1 делает сервис недоступным извне контейнера.
ENV HOST=0.0.0.0
ENV PORT=8080

WORKDIR /app

USER appuser

EXPOSE 8080

# Настройки (DATABASE_DSN, MINIO_*, и т.д.) задаются через environment в docker-compose или .env — см. .env.example
CMD ["server"]
