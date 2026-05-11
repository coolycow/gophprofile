package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/coolycow/gophprofile/internal/logger"
	flag "github.com/spf13/pflag"
)

type ConfigWorker struct {
	Host          string `env:"HOST" json:"host,omitempty"`                     // IP адрес сервера
	Port          int    `env:"PORT" json:"port,omitempty"`                     // Порт сервера
	LogLevel      string `env:"LOG_LEVEL" json:"log_level,omitempty"`           // Уровень логирования
	DatabaseDSN   string `env:"DATABASE_DSN" json:"database_dsn,omitempty"`     // DSN для подключения к БД
	RunMigrations bool   `env:"RUN_MIGRATIONS" json:"run_migrations,omitempty"` // Флаг для запуска миграций
	EnableHTTPS   bool   `env:"ENABLE_HTTPS" json:"enable_https,omitempty"`     // Флаг для включения HTTPS
	TLSCertFile   string `env:"TLS_CERT_FILE" json:"tls_cert_file,omitempty"`   // Файл сертификата для HTTPS
	TLSKeyFile    string `env:"TLS_KEY_FILE" json:"tls_key_file,omitempty"`     // Файл ключа для HTTPS
	TrustedSubnet string `env:"TRUSTED_SUBNET" json:"trusted_subnet,omitempty"` // Подсеть для доступа к статистике
	Config        string `env:"CONFIG" json:"config,omitempty"`                 // Путь к файлу конфигурации

	MinioEndpoint   string `env:"MINIO_ENDPOINT" json:"minio_endpoint,omitempty"`       // Endpoint MinIO
	MinioAccessKey  string `env:"MINIO_ACCESS_KEY" json:"minio_access_key,omitempty"`   // Access Key MinIO
	MinioSecretKey  string `env:"MINIO_SECRET_KEY" json:"minio_secret_key,omitempty"`   // Secret Key MinIO
	MinioBucketName string `env:"MINIO_BUCKET_NAME" json:"minio_bucket_name,omitempty"` // Имя бакета MinIO
	MinioUseSSL     bool   `env:"MINIO_USE_SSL" json:"minio_use_ssl,omitempty"`         // Использовать SSL для MinIO
}

// fileConfig — JSON-файл; указатели задают поля, явно присутствующие в файле.
// Ключ "config" в файле не разбираем (путь к файлу только из -c / CONFIG).
type fileConfigWorker struct {
	Host            *string `json:"host"`              // IP адрес сервера
	Port            *int    `json:"port"`              // Порт сервера
	LogLevel        *string `json:"log_level"`         // Уровень логирования
	DatabaseDSN     *string `json:"database_dsn"`      // DSN для подключения к БД
	RunMigrations   *bool   `json:"run_migrations"`    // Флаг для запуска миграций
	EnableHTTPS     *bool   `json:"enable_https"`      // Флаг для включения HTTPS
	TLSCertFile     *string `json:"tls_cert_file"`     // Файл сертификата для HTTPS
	TLSKeyFile      *string `json:"tls_key_file"`      // Файл ключа для HTTPS
	TrustedSubnet   *string `json:"trusted_subnet"`    // Подсеть для доступа к статистике
	Config          *string `json:"config"`            // Путь к файлу конфигурации
	MinioEndpoint   *string `json:"minio_endpoint"`    // Endpoint MinIO
	MinioAccessKey  *string `json:"minio_access_key"`  // Access Key MinIO
	MinioSecretKey  *string `json:"minio_secret_key"`  // Secret Key MinIO
	MinioBucketName *string `json:"minio_bucket_name"` // Имя бакета MinIO
	MinioUseSSL     *bool   `json:"minio_use_ssl"`     // Использовать SSL для MinIO
}

// PrintWorkerConfig записывает полный дамп настроек одной строкой в лог (вызывать после logger.Initialize).
func (c *ConfigWorker) PrintWorkerConfig() {
	var b strings.Builder

	// Формируем строку с настройками
	fmt.Fprintf(&b, "Host=%s Port=%d ", c.Host, c.Port)
	fmt.Fprintf(&b, "LogLevel=%s RunMigrations=%t; ", c.LogLevel, c.RunMigrations)
	fmt.Fprintf(&b, "EnableHTTPS=%t TLSCertFile=%s TLSKeyFile=%s TrustedSubnet=%s Config=%s; ",
		c.EnableHTTPS, c.TLSCertFile, c.TLSKeyFile, c.TrustedSubnet, c.Config)
	fmt.Fprintf(&b, "MinioEndpoint=%s MinioAccessKey=%s MinioSecretKey=%s MinioBucketName=%s MinioUseSSL=%t; ", c.MinioEndpoint, c.MinioAccessKey, c.MinioSecretKey, c.MinioBucketName, c.MinioUseSSL)
	// Выводим настройки в лог
	logger.Log.Info(b.String())
}

// InitConfigWorker возвращает настройки и ошибку если парсинг аргументов не удался.
// Порядок приоритета: значения по умолчанию → JSON-файл → переменные окружения → флаги.
func InitConfigWorker() (*ConfigWorker, error) {
	args := os.Args[1:]

	// Получаем настройки из флагов
	flagCfg, fs, err := parseWorkerFlags(args)
	if err != nil {
		return nil, err
	}

	// Получаем настройки из переменных окружения
	configPath := strings.TrimSpace(flagCfg.Config)
	if !fs.Changed("config") {
		if v, ok := os.LookupEnv("CONFIG"); ok {
			configPath = strings.TrimSpace(v)
		}
	}

	// Получаем настройки из файла конфигурации
	cfg := defaultConfigWorker()
	_, configEnvSet := os.LookupEnv("CONFIG")
	explicitConfigPath := fs.Changed("config") || configEnvSet

	if configPath != "" {
		if err := mergeConfigWorkerFromFile(&cfg, configPath); err != nil {
			if errors.Is(err, os.ErrNotExist) && !explicitConfigPath {
				cfg = defaultConfigWorker()
			} else {
				return nil, err
			}
		}
	}

	cfg.Config = configPath

	// Применяем настройки из переменных окружения
	if _, err := applyEnvToConfigWorker(&cfg, fs.Changed("config")); err != nil {
		return nil, err
	}

	applyExplicitWorkerFlags(&cfg, flagCfg, fs)

	// Проверяем настройки на корректность
	var errs []error
	if ts := strings.TrimSpace(cfg.TrustedSubnet); ts != "" {
		if _, _, err := net.ParseCIDR(ts); err != nil {
			errs = append(errs, fmt.Errorf("trusted_subnet: %w", err))
		}
	}

	return &cfg, errors.Join(errs...)
}

// initConfigWorkerWithEnv получение настроек из переменных окружения.
func initConfigWorkerWithEnv(config *ConfigWorker) (*ConfigWorker, error) {
	return applyEnvToConfigWorker(config, false)
}

// applyEnvToConfigWorker применяет переменные окружения. Если skipConfigFromEnv, CONFIG не трогаем
// (путь к файлу задан явно флагом -c).
func applyEnvToConfigWorker(config *ConfigWorker, skipConfigFromEnv bool) (*ConfigWorker, error) {
	if host, present := os.LookupEnv("HOST"); present {
		config.Host = host
	}

	if err := parseIntFromWorkerEnv(config, "PORT",
		func(c *ConfigWorker, v int) { c.Port = v }); err != nil {
		return nil, err
	}

	if logLevel, present := os.LookupEnv("LOG_LEVEL"); present {
		config.LogLevel = logLevel
	}

	if databaseDSN, present := os.LookupEnv("DATABASE_DSN"); present {
		config.DatabaseDSN = databaseDSN
	}

	if runMigrations, present := os.LookupEnv("RUN_MIGRATIONS"); present {
		config.RunMigrations, _ = strconv.ParseBool(runMigrations)
	}

	if enableHTTPS, present := os.LookupEnv("ENABLE_HTTPS"); present {
		config.EnableHTTPS, _ = strconv.ParseBool(enableHTTPS)
	}

	if certFile, present := os.LookupEnv("TLS_CERT_FILE"); present {
		config.TLSCertFile = certFile
	}

	if keyFile, present := os.LookupEnv("TLS_KEY_FILE"); present {
		config.TLSKeyFile = keyFile
	}

	if trustedSubnet, present := os.LookupEnv("TRUSTED_SUBNET"); present {
		config.TrustedSubnet = trustedSubnet
	}

	if minioEndpoint, present := os.LookupEnv("MINIO_ENDPOINT"); present {
		config.MinioEndpoint = minioEndpoint
	}
	if minioAccessKey, present := os.LookupEnv("MINIO_ACCESS_KEY"); present {
		config.MinioAccessKey = minioAccessKey
	}
	if minioSecretKey, present := os.LookupEnv("MINIO_SECRET_KEY"); present {
		config.MinioSecretKey = minioSecretKey
	}
	if minioBucketName, present := os.LookupEnv("MINIO_BUCKET_NAME"); present {
		config.MinioBucketName = minioBucketName
	}
	if minioUseSSL, present := os.LookupEnv("MINIO_USE_SSL"); present {
		config.MinioUseSSL, _ = strconv.ParseBool(minioUseSSL)
	}

	if !skipConfigFromEnv {
		if configFile, present := os.LookupEnv("CONFIG"); present {
			config.Config = strings.TrimSpace(configFile)
		}
	}

	return config, nil
}

// InitConfigWorkerWithArgs инициализация с переданными аргументами (только флаги; как раньше для тестов).
func InitConfigWorkerWithArgs(args []string) (*ConfigWorker, error) {
	cfg, _, err := parseWorkerFlags(args)
	return cfg, err
}

// parseWorkerFlags парсит флаги для сервера
func parseWorkerFlags(args []string) (*ConfigWorker, *flag.FlagSet, error) {
	var config ConfigWorker

	flagSet := flag.NewFlagSet("main", flag.ContinueOnError)

	// Флаги для сервера
	flagSet.StringVarP(&config.Host, "host", "h", "127.0.0.1", "server host")
	flagSet.IntVarP(&config.Port, "port", "p", 8080, "server port")

	// Флаги для логирования
	flagSet.StringVarP(&config.LogLevel, "log-level", "l", "info", "log level")

	// Флаги для БД
	flagSet.StringVarP(&config.DatabaseDSN, "database-dsn", "d", getDefaultDatabaseDSN(), "database DSN")
	flagSet.BoolVarP(&config.RunMigrations, "run-migrations", "r", false, "run migrations")

	// Флаги для HTTPS
	flagSet.BoolVarP(&config.EnableHTTPS, "enable-https", "s", false, "enable HTTPS")
	flagSet.StringVarP(&config.TLSCertFile, "tls-cert-file", "t", getDefaultTLSCertFile(), "TLS certificate file (PEM), for HTTPS")
	flagSet.StringVarP(&config.TLSKeyFile, "tls-key-file", "k", getDefaultTLSKeyFile(), "TLS private key file (PEM), for HTTPS")

	// Флаги для конфигурации
	flagSet.StringVarP(&config.Config, "config", "c", getDefaultConfigFile(), "config file")

	// Флаги для статистики (без короткого имени: «s» занят enable-https)
	flagSet.StringVar(&config.TrustedSubnet, "trusted-subnet", "", "trusted CIDR for GET /api/internal/stats (X-Real-IP)")

	// Флаги для MinIO
	flagSet.StringVar(&config.MinioEndpoint, "minio-endpoint", "", "MinIO endpoint")
	flagSet.StringVar(&config.MinioAccessKey, "minio-access-key", "", "MinIO access key")
	flagSet.StringVar(&config.MinioSecretKey, "minio-secret-key", "", "MinIO secret key")
	flagSet.StringVar(&config.MinioBucketName, "minio-bucket-name", "", "MinIO bucket name")
	flagSet.BoolVar(&config.MinioUseSSL, "minio-use-ssl", false, "Use SSL for MinIO")

	// Парсим флаги
	err := flagSet.Parse(args)
	if err != nil {
		return nil, nil, err
	}

	return &config, flagSet, nil
}

// defaultConfigWorker возвращает конфигурацию сервера по умолчанию
func defaultConfigWorker() ConfigWorker {
	return ConfigWorker{
		Host:            "127.0.0.1",
		Port:            8080,
		LogLevel:        "info",
		DatabaseDSN:     getDefaultDatabaseDSN(),
		RunMigrations:   false,
		EnableHTTPS:     false,
		TLSCertFile:     getDefaultTLSCertFile(),
		TLSKeyFile:      getDefaultTLSKeyFile(),
		TrustedSubnet:   "",
		Config:          "",
		MinioEndpoint:   "",
		MinioAccessKey:  "",
		MinioSecretKey:  "",
		MinioBucketName: "",
		MinioUseSSL:     false,
	}
}

// mergeConfigWorkerFromFile объединяет конфигурацию сервера с данными из файла
func mergeConfigWorkerFromFile(cfg *ConfigWorker, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var fc fileConfigWorker
	if err := json.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("invalid config file %s: %w", path, err)
	}

	if fc.Host != nil {
		cfg.Host = *fc.Host
	}

	if fc.Port != nil {
		cfg.Port = *fc.Port
	}

	if fc.LogLevel != nil {
		cfg.LogLevel = *fc.LogLevel
	}
	if fc.DatabaseDSN != nil {
		cfg.DatabaseDSN = *fc.DatabaseDSN
	}
	if fc.RunMigrations != nil {
		cfg.RunMigrations = *fc.RunMigrations
	}
	if fc.EnableHTTPS != nil {
		cfg.EnableHTTPS = *fc.EnableHTTPS
	}
	if fc.TLSCertFile != nil {
		cfg.TLSCertFile = *fc.TLSCertFile
	}
	if fc.TLSKeyFile != nil {
		cfg.TLSKeyFile = *fc.TLSKeyFile
	}
	if fc.TrustedSubnet != nil {
		cfg.TrustedSubnet = *fc.TrustedSubnet
	}
	if fc.MinioEndpoint != nil {
		cfg.MinioEndpoint = *fc.MinioEndpoint
	}
	if fc.MinioAccessKey != nil {
		cfg.MinioAccessKey = *fc.MinioAccessKey
	}
	if fc.MinioSecretKey != nil {
		cfg.MinioSecretKey = *fc.MinioSecretKey
	}
	if fc.MinioBucketName != nil {
		cfg.MinioBucketName = *fc.MinioBucketName
	}
	if fc.MinioUseSSL != nil {
		cfg.MinioUseSSL = *fc.MinioUseSSL
	}

	return nil
}

// applyExplicitWorkerFlags применяет переданные флаги к конфигурации сервера
func applyExplicitWorkerFlags(dst *ConfigWorker, src *ConfigWorker, fs *flag.FlagSet) {
	if fs.Changed("host") {
		dst.Host = src.Host
	}
	if fs.Changed("port") {
		dst.Port = src.Port
	}
	if fs.Changed("log-level") {
		dst.LogLevel = src.LogLevel
	}
	if fs.Changed("database-dsn") {
		dst.DatabaseDSN = src.DatabaseDSN
	}
	if fs.Changed("run-migrations") {
		dst.RunMigrations = src.RunMigrations
	}
	if fs.Changed("enable-https") {
		dst.EnableHTTPS = src.EnableHTTPS
	}
	if fs.Changed("tls-cert-file") {
		dst.TLSCertFile = src.TLSCertFile
	}
	if fs.Changed("tls-key-file") {
		dst.TLSKeyFile = src.TLSKeyFile
	}
	if fs.Changed("trusted-subnet") {
		dst.TrustedSubnet = src.TrustedSubnet
	}
	if fs.Changed("address") {
		dst.Host = src.Host
		dst.Port = src.Port
	}
	if fs.Changed("config") {
		dst.Config = strings.TrimSpace(src.Config)
	}
	if fs.Changed("minio-endpoint") {
		dst.MinioEndpoint = src.MinioEndpoint
	}
	if fs.Changed("minio-access-key") {
		dst.MinioAccessKey = src.MinioAccessKey
	}
	if fs.Changed("minio-secret-key") {
		dst.MinioSecretKey = src.MinioSecretKey
	}
	if fs.Changed("minio-bucket-name") {
		dst.MinioBucketName = src.MinioBucketName
	}
	if fs.Changed("minio-use-ssl") {
		dst.MinioUseSSL = src.MinioUseSSL
	}
}

// parseIntFromWorkerEnv парсит int-значение из переменной окружения и устанавливает его в поле конфигурации
func parseIntFromWorkerEnv(config *ConfigWorker, envKey string, setter func(*ConfigWorker, int)) error {
	if value, present := os.LookupEnv(envKey); present {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid env %s %s", envKey, value)
		}
		setter(config, intValue)
	}
	return nil
}
