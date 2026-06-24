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
GET /health          # readiness (обратная совместимость)
GET /health/live     # liveness probe (Kubernetes)
GET /health/ready    # readiness probe (Kubernetes)
```

## API документация (OpenAPI / Swagger)

- Спецификация (OpenAPI 3): [docs/openapi.yaml](docs/openapi.yaml)
- Swagger 2 (генерируется swag): [docs/swagger/swagger.yaml](docs/swagger/swagger.yaml)
- Swagger UI (dev или `SWAGGER_ENABLED=true`): http://localhost:8333/swagger/index.html

Генерация кода и Swagger 2 из аннотаций:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/server/main.go -o docs/swagger --outputTypes go,yaml
```

После изменения handlers сверьте и при необходимости обновите `docs/openapi.yaml` (OpenAPI 3 для IDE и внешних клиентов).

## Архитектура

См. [docs/architecture.md](docs/architecture.md) — диаграммы компонентов, потоков данных и K8s-развёртывания.

## Kubernetes (Rancher Desktop и другие кластеры)

### Предварительные требования

- Kubernetes-кластер (Rancher Desktop, minikube, kind и т.д.)
- `kubectl`, `helm`, `docker`
- Ingress Controller:
  - **Rancher Desktop** — уже установлен **Traefik** (`ingressClassName: traefik` в `values-local.yaml`)
  - **Другие кластеры** — nginx:
  ```bash
  helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
  helm install ingress-nginx ingress-nginx/ingress-nginx -n ingress-nginx --create-namespace
  kubectl label namespace ingress-nginx name=ingress-nginx --overwrite
  ```
- metrics-server (для HPA; в Rancher Desktop обычно уже установлен)
- Prometheus Operator (для ServiceMonitor, опционально):
  ```bash
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
  helm install kube-prometheus prometheus-community/kube-prometheus-stack -n monitoring --create-namespace
  ```

### 1. Сборка Docker-образов

**Важно:** если Kubernetes в **Rancher Desktop**, а `docker build` идёт в **Docker Desktop** — кластер не увидит образы (`ErrImageNeverPull`, `helm install` «зависает» на migration Job).

Соберите образы в Docker Rancher Desktop:

```powershell
powershell -ExecutionPolicy Bypass -File deploy/scripts/build-images-rancher.ps1
```

Или вручную через `rdctl shell`:

```bash
rdctl shell -- docker build -f /mnt/i/practicum/gophprofile/docker/server/Dockerfile -t gophprofile/server:latest /mnt/i/practicum/gophprofile
rdctl shell -- docker build -f /mnt/i/practicum/gophprofile/docker/worker/Dockerfile -t gophprofile/worker:latest /mnt/i/practicum/gophprofile
rdctl shell -- docker build -f /mnt/i/practicum/gophprofile/docker/client/Dockerfile -t gophprofile/client:latest /mnt/i/practicum/gophprofile
```

Если Docker и Kubernetes в одном Rancher Desktop, достаточно обычного `docker build` (см. ниже).

```bash
docker build -f docker/server/Dockerfile -t gophprofile/server:latest .
docker build -f docker/worker/Dockerfile -t gophprofile/worker:latest .
docker build -f docker/client/Dockerfile -t gophprofile/client:latest .
```

`values-local.yaml` использует `imagePullPolicy: Never` — образ должен уже быть на ноде k3s.

### 2. Инфраструктура (PostgreSQL, RabbitMQ, MinIO)

```bash
kubectl apply -f deploy/k8s/infra/namespaces.yaml
kubectl apply -f deploy/k8s/infra/postgres/
kubectl apply -f deploy/k8s/infra/rabbitmq/
kubectl apply -f deploy/k8s/infra/minio/
kubectl wait --for=condition=ready pod -l app=postgres -n database --timeout=120s
kubectl wait --for=condition=ready pod -l app=rabbitmq -n database --timeout=180s
kubectl wait --for=condition=ready pod -l app=minio -n database --timeout=120s
kubectl apply -f deploy/k8s/infra/minio/init-bucket-job.yaml
```

> **PostgreSQL 18:** том монтируется в `/var/lib/postgresql`. Если pod `postgres-0` в CrashLoopBackOff после первой неудачной попытки, удалите PVC и пересоздайте StatefulSet: `kubectl delete pvc data-postgres-0 -n database && kubectl rollout restart statefulset/postgres -n database`

### 3. Деплой приложения (Helm)

Helm входит в состав Rancher Desktop, но может отсутствовать в PATH. В PowerShell:

```powershell
$env:Path += ";C:\Program Files\Rancher Desktop\resources\resources\win32\bin"
helm version
```

```bash
helm install gophprofile deploy/helm/gophprofile \
  -n gophprofile --create-namespace \
  -f deploy/helm/gophprofile/values-local.yaml \
  --timeout 5m
```

> Если команда «зависает» без вывода — Helm ждёт migration Job. Проверьте: `kubectl get pods -n gophprofile`. Частые причины: `ErrImageNeverPull` (образы собраны не в Rancher Docker) или `secret not found` (исправлено в chart).

> **ServiceMonitor** в `values-local.yaml` отключён по умолчанию. Для мониторинга установите kube-prometheus-stack и включите `monitoring.serviceMonitor.enabled=true`.

Обновление релиза:
```bash
helm upgrade gophprofile deploy/helm/gophprofile -n gophprofile -f deploy/helm/gophprofile/values-local.yaml
```

Production-переопределения: `deploy/helm/gophprofile/values-prod.yaml`

### 4. Доступ через Ingress

Добавьте в hosts (Windows: `C:\Windows\System32\drivers\etc\hosts`):
```
127.0.0.1 gophprofile.local
```

Проверка:
```bash
curl http://gophprofile.local/health
curl http://gophprofile.local/health/ready
```

### Чек-лист приёмки (Kubernetes)

| Критерий | Команда / действие |
|----------|-------------------|
| Все поды Running | `kubectl get pods -n gophprofile -n database` |
| Health probes | `kubectl describe pod -n gophprofile -l app=gophprofile-server` |
| Ingress | `curl http://gophprofile.local/health` |
| HPA | `kubectl get hpa -n gophprofile` |
| Метрики | Prometheus UI → Targets (ServiceMonitor) |
| Non-root | `kubectl exec -n gophprofile deploy/gophprofile-server -- id` → uid 65532 |
| Secrets | `kubectl get secret -n gophprofile` — credentials не в ConfigMap |
| Graceful shutdown | `kubectl delete pod -n gophprofile -l app=gophprofile-server` — без 502 при rolling update |

### Helm Chart: основные ресурсы

- Deployments: server, worker, client
- Services + Ingress
- ConfigMap / Secret
- HPA (CPU/RAM)
- ServiceMonitor (Prometheus)
- NetworkPolicy, RBAC, ServiceAccount
- Migration Job (Helm hook)

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

### Мониторинг и алерты в Kubernetes

- ServiceMonitor создаётся Helm-чартом (`monitoring.serviceMonitor.enabled: true`)
- Импортируйте дашборд `deploy/observability/grafana/dashboards/gophprofile-overview.json` в Grafana
- Правила алертов: `deploy/observability/prometheus/alerts.yml` (высокий error rate, latency, падение worker consumer)

## Production-ready возможности

- **Graceful shutdown** — HTTP-сервер завершается за 30 с, RabbitMQ закрывается явно
- **Circuit breaker** — MinIO, RabbitMQ, PostgreSQL (`internal/resilience`)
- **Rate limiting** — token bucket по IP (`RATE_LIMIT_*`)
- **Resource limits** — в Helm values и Docker/K8s manifests
- **Health probes** — `/health/live`, `/health/ready`

## Тестирование

Для запуска теста из корня необходимо вызвать команду:
```bash
go test -v ./...
```

Для того, чтобы узнать процент покрытия тестами используется команда:
```bash
go test -cover ./...