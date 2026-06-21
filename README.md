# GophProfile
Микросервис для управления аватарками пользователей — GophProfile.

## Для чего нужен
Пользователь загружает свою фотографию в GophProfile один раз, а дальше любые сторонние платформы (блоги, форумы, сервисы комментариев и др.) могут запросить аватарку по его email. Если пользователь с таким email существует, сервис отдаёт его аватар. В противном случае возвращается стандартное изображение-заглушка.

## Фронтенд
Ссылка на готовый фронтенд: [https://github.com/Yandex-Practicum/go-avatar-service-template](https://github.com/Yandex-Practicum/go-avatar-service-template).

Для запуска клиента: http://localhost:8332

## База данных и миграции
Используется контейнер `postgres` с именованным хранилищем.

## Хранилище изображений
Для хранения файлов используется MinIO.
По умолчанию доступ с хоста:
* API: http://127.0.0.1:9002
* WebUI: http://127.0.0.1:9003

Внутри Docker-сети сервисы обращаются к `minio:9000`. Проброс портов настраивается через `MINIO_API_PUBLISH_PORT` и `MINIO_CONSOLE_PUBLISH_PORT` в `.env`.

Пользователь и пароль задаются в `.env` (переменные `MINIO_ROOT_USER` и `MINIO_ROOT_PASSWORD`).

Название корзины, в которой хранятся аватарки пользователей, задаётся в переменной `MINIO_BUCKET_NAME`.

## RabbitMQ
WebUI: http://localhost:15673 (порт проброса на хост; внутри Docker — `rabbitmq:5672`)

Пользователь и пароль задаются в `.env` (переменные `RABBITMQ_DEFAULT_USER` и `RABBITMQ_DEFAULT_PASS`).

Если порты `5672`/`15672` на машине заняты, задайте в `.env`: `RABBITMQ_PUBLISH_PORT` и `RABBITMQ_MANAGEMENT_PUBLISH_PORT`.

### Миниатюры
Миниатюры генерируются в двух размерах: `100x100` и `300x300`. Формат миниатюр - `JPEG`.

В запросах `/avatars/:avatar_id` и `/users/:user_id/avatar` поддерживаются два параметра:
* `size` - размер требуемой миниатюры, допустимы значения `100x100`, `300x300` и `original`;
* `format` - требуемый формат изображения, допустимые значения `jpeg`, `png` и `webp`. Генерация `png` и `webp` происходит в момент выполнения запроса.

## Сборка и запуск
Обязательно необходимо создать:
1. ENV-файл `.env` в корне (задаёт настройки для большей части проекта), например:
```bash
cp .env.example .env
```
2. ENV-файл `.env` в `docker/postgres` (задаёт настроки для контейнера Postgres), например:
```bash
cp docker/postgres/.env.example docker/postgres/.env
```

Для разработки и тестирования содержимое файлов `.env` можно не изменять, но на реальном сервере все логины/пароли должны быть установлены в соответствии с общепринятыми нормами.

Для сборки:
```bash
docker compose build
```

Для запуска:
```bash
docker compose up -d
```

## Запросы
### Загрузка аватарки
```bash
POST /api/v1/avatars
Content-Type: multipart/form-data
Headers: X-User-ID: string
```

### Получение аватарки
```bash
GET /api/v1/avatars/{avatar_id}
GET /api/v1/users/{user_id}/avatar
```

### Удаление аватарки
```bash
DELETE /api/v1/avatars/{avatar_id}
DELETE /api/v1/users/{user_id}/avatar
Headers: X-User-ID: string
```

### Получение метаданных аватарки
```bash
GET /api/v1/avatars/{avatar_id}/metadata
```

### Список аватарок пользователя
```bash
GET /api/v1/users/{user_id}/avatars
```

### Проверка работоспособности
```bash
GET /health
```

## Observability

Стек наблюдаемости: **OpenTelemetry** (трейсы), **Prometheus** (метрики), **Grafana Loki** (логи), **Jaeger** (UI трейсов), **Grafana** (дашборды), **Alertmanager** (алерты).

| Сервис | URL |
|--------|-----|
| Grafana | http://localhost:3000 (admin/admin по умолчанию) |
| Jaeger | http://localhost:16686 |
| Prometheus | http://localhost:9090 |
| Alertmanager | http://localhost:9093 |
| Loki | http://localhost:3100 |

Метрики приложения:
- Server: http://localhost:9091/metrics
- Worker: http://localhost:9092/metrics

Переменные окружения (см. `.env.example`):
- `OTEL_ENABLED` — включить экспорт трейсов
- `OTEL_EXPORTER_OTLP_ENDPOINT` — адрес Jaeger OTLP (по умолчанию `jaeger:4317`)
- `OTEL_SERVICE_NAME` — имя сервиса в трейсах
- `METRICS_ADDR` — адрес HTTP `/metrics` (по умолчанию `:9090`)
- `LOG_FORMAT=json` — JSON-логи для Loki

### Проверка end-to-end

1. `docker compose up -d --build`
2. Загрузите аватар через API или UI (http://localhost:8332)
3. **Jaeger**: найдите trace `gophprofile-server` → span `upload_avatar` → `rabbitmq.publish` → `gophprofile-worker` → `worker.process_avatar`
4. **Grafana → Explore → Loki**: `{service="server"} | json` — фильтр по `trace_id`
5. **Grafana → Dashboards → GophProfile Overview**: RED-метрики, KPI загрузок, логи ошибок

## Тестирование

Для запуска теста из корня необходимо вызвать команду:
```bash
go test -v ./...
```

Для того, чтобы узнать процент покрытия тестами используется команда:
```bash
go test -cover ./...