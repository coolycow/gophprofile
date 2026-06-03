package observability

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var registry = prometheus.NewRegistry()

func init() {
	registry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
}

var (
	HTTPRequestsTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "route", "status"},
	)

	HTTPRequestDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)

	AvatarsUploadsTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "avatars_uploads_total",
			Help: "Total number of avatar uploads",
		},
		[]string{"status", "user_id"},
	)

	AvatarsUploadDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "avatars_upload_duration_seconds",
			Help:    "Avatar upload duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)

	AvatarsStorageBytes = promauto.With(registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "avatars_storage_bytes",
			Help: "Total storage used by avatars per user",
		},
		[]string{"user_id"},
	)

	AvatarJobsProcessedTotal = promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "avatar_jobs_processed_total",
			Help: "Total number of avatar jobs processed by worker",
		},
		[]string{"job_type", "status"},
	)

	AvatarJobDuration = promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "avatar_job_duration_seconds",
			Help:    "Avatar job processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"job_type", "status"},
	)

	DBConnectionsOpen = promauto.With(registry).NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_open",
			Help: "Number of open database connections",
		},
	)

	DBConnectionsInUse = promauto.With(registry).NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_in_use",
			Help: "Number of database connections currently in use",
		},
	)
)

// StartMetricsServer запускает отдельный HTTP-сервер с эндпоинтом /metrics.
func StartMetricsServer(ctx context.Context, addr string) (*http.Server, error) {
	addr = trimMetricsAddr(addr)
	if addr == "" {
		return nil, nil
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			LoggerFromContext(ctx).Error("metrics server error", "error", err)
		}
	}()

	return srv, nil
}

func trimMetricsAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" || addr == "off" || addr == "disabled" {
		return ""
	}
	return addr
}

// StartDBStatsCollector периодически обновляет метрики пула соединений БД.
func StartDBStatsCollector(ctx context.Context, db *sql.DB, interval time.Duration) {
	if db == nil {
		return
	}
	if interval <= 0 {
		interval = 15 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := db.Stats()
				DBConnectionsOpen.Set(float64(stats.OpenConnections))
				DBConnectionsInUse.Set(float64(stats.InUse))
			}
		}
	}()
}

// ObserveUpload записывает метрики загрузки аватара.
func ObserveUpload(userID, status string, duration time.Duration, sizeBytes int64) {
	AvatarsUploadsTotal.WithLabelValues(status, userID).Inc()
	AvatarsUploadDuration.WithLabelValues(status).Observe(duration.Seconds())
	if status == "success" && sizeBytes > 0 {
		AvatarsStorageBytes.WithLabelValues(userID).Add(float64(sizeBytes))
	}
}

// ObserveUploadDelete уменьшает gauge хранилища при удалении.
func ObserveUploadDelete(userID string, sizeBytes int64) {
	if sizeBytes > 0 {
		AvatarsStorageBytes.WithLabelValues(userID).Sub(float64(sizeBytes))
	}
}

// ObserveJob записывает метрики обработки задания воркером.
func ObserveJob(jobType, status string, duration time.Duration) {
	AvatarJobsProcessedTotal.WithLabelValues(jobType, status).Inc()
	AvatarJobDuration.WithLabelValues(jobType, status).Observe(duration.Seconds())
}

// MetricsRegistry возвращает prometheus registry приложения (для тестов).
func MetricsRegistry() *prometheus.Registry {
	return registry
}

// ShutdownMetricsServer корректно останавливает metrics HTTP server.
func ShutdownMetricsServer(ctx context.Context, srv *http.Server) error {
	if srv == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// MustRegister дополнительных коллекторов (exporters и т.п.).
func MustRegister(collectors ...prometheus.Collector) {
	registry.MustRegister(collectors...)
}

// FormatMetricsAddr нормализует адрес metrics server.
func FormatMetricsAddr(addr string) string {
	if addr == "" {
		return ":9090"
	}
	return addr
}

// ValidateMetricsAddr проверяет, что адрес задан корректно.
func ValidateMetricsAddr(addr string) error {
	addr = trimMetricsAddr(addr)
	if addr == "" {
		return nil
	}
	if addr[0] != ':' && !strings.Contains(addr, ":") {
		return fmt.Errorf("metrics_addr: invalid address %q", addr)
	}
	return nil
}
