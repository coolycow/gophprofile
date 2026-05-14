package service

import (
	"net/url"
	"strings"

	"github.com/coolycow/gophprofile/internal/config"
)

// MinioObjectPublicURL path-style публичный URL объекта: {base}/{bucket}/{objectKey}.
// Если cfg.MinioPublicBaseURL задан — используется как корень (без завершающего /).
// Иначе корень: http(s)://{MinioEndpoint}.
func MinioObjectPublicURL(cfg *config.ConfigServer, objectKey string) string {
	if cfg == nil {
		return ""
	}

	return minioObjectPublicURL(
		cfg.MinioEndpoint,
		cfg.MinioUseSSL,
		cfg.MinioBucketName,
		objectKey,
		cfg.MinioPublicBaseURL,
	)
}

// minioObjectPublicURL генерирует публичный URL объекта в S3
func minioObjectPublicURL(endpoint string, useSSL bool, bucket, objectKey, publicBaseOverride string) string {
	objectKey = strings.TrimPrefix(strings.TrimSpace(objectKey), "/")
	bucket = strings.TrimSpace(bucket)
	if objectKey == "" || bucket == "" {
		return ""
	}

	pathPart := "/" + pathEscapeURLPath(bucket+"/"+objectKey)

	base := strings.TrimRight(strings.TrimSpace(publicBaseOverride), "/")
	if base == "" {
		ep := strings.TrimSpace(endpoint)
		ep = strings.TrimPrefix(ep, "http://")
		ep = strings.TrimPrefix(ep, "https://")
		ep = strings.TrimSuffix(ep, "/")
		if ep == "" {
			return ""
		}
		scheme := "http"
		if useSSL {
			scheme = "https"
		}
		base = scheme + "://" + ep
	}

	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.TrimSuffix(base, "/") + pathPart
	}
	ref, err := url.Parse(pathPart)
	if err != nil {
		return strings.TrimSuffix(base, "/") + pathPart
	}
	return u.ResolveReference(ref).String()
}

// pathEscapeURLPath экранирует путь URL
func pathEscapeURLPath(p string) string {
	// Обрезаем пробелы
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}

	// Разделяем путь на части по разделителю "/"
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}

	return strings.Join(parts, "/")
}
