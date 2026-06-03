package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/coolycow/gophprofile/internal/logger"
	flag "github.com/spf13/pflag"
)

type ConfigWorker struct {
	LogLevel      string `env:"LOG_LEVEL" json:"log_level,omitempty"`           // Уровень логирования
	DatabaseDSN   string `env:"DATABASE_DSN" json:"database_dsn,omitempty"`     // DSN для подключения к БД
	RunMigrations bool   `env:"RUN_MIGRATIONS" json:"run_migrations,omitempty"` // Флаг для запуска миграций
	Config        string `env:"CONFIG" json:"config,omitempty"`                 // Путь к файлу конфигурации

	RabbitMQHost     string `env:"RABBITMQ_HOST" json:"rabbitmq_host,omitempty"`         // Хост RabbitMQ
	RabbitMQPort     int    `env:"RABBITMQ_PORT" json:"rabbitmq_port,omitempty"`         // Порт AMQP
	RabbitMQUser     string `env:"RABBITMQ_USER" json:"rabbitmq_user,omitempty"`         // Пользователь RabbitMQ
	RabbitMQPassword string `env:"RABBITMQ_PASSWORD" json:"rabbitmq_password,omitempty"` // Пароль RabbitMQ
	RabbitMQVHost    string `env:"RABBITMQ_VHOST" json:"rabbitmq_vhost,omitempty"`       // Виртуальный хост

	MaxFileSize int `env:"MAX_FILE_SIZE" json:"max_file_size,omitempty"` // Максимальный размер входного файла (байты)

	MinioEndpoint   string `env:"MINIO_ENDPOINT" json:"minio_endpoint,omitempty"`       // Endpoint MinIO
	MinioAccessKey  string `env:"MINIO_ACCESS_KEY" json:"minio_access_key,omitempty"`   // Access Key MinIO
	MinioSecretKey  string `env:"MINIO_SECRET_KEY" json:"minio_secret_key,omitempty"`   // Secret Key MinIO
	MinioBucketName string `env:"MINIO_BUCKET_NAME" json:"minio_bucket_name,omitempty"` // Имя бакета MinIO
	MinioUseSSL     bool   `env:"MINIO_USE_SSL" json:"minio_use_ssl,omitempty"`         // Использовать SSL для MinIO

	Observability ObservabilityConfig
}

// fileConfig — JSON-файл; указатели задают поля, явно присутствующие в файле.
// Ключ "config" в файле не разбираем (путь к файлу только из -c / CONFIG).
type fileConfigWorker struct {
	LogLevel         *string `json:"log_level"`         // Уровень логирования
	DatabaseDSN      *string `json:"database_dsn"`      // DSN для подключения к БД
	RunMigrations    *bool   `json:"run_migrations"`    // Флаг для запуска миграций
	Config           *string `json:"config"`            // Путь к файлу конфигурации
	RabbitMQHost     *string `json:"rabbitmq_host"`     // Хост RabbitMQ
	RabbitMQPort     *int    `json:"rabbitmq_port"`     // Порт AMQP
	RabbitMQUser     *string `json:"rabbitmq_user"`     // Пользователь RabbitMQ
	RabbitMQPassword *string `json:"rabbitmq_password"` // Пароль RabbitMQ
	RabbitMQVHost    *string `json:"rabbitmq_vhost"`    // Виртуальный хост
	MaxFileSize      *int    `json:"max_file_size"`     // Максимальный размер файла (байты)
	MinioEndpoint    *string `json:"minio_endpoint"`    // Endpoint MinIO
	MinioAccessKey   *string `json:"minio_access_key"`  // Access Key MinIO
	MinioSecretKey   *string `json:"minio_secret_key"`  // Secret Key MinIO
	MinioBucketName  *string `json:"minio_bucket_name"` // Имя бакета MinIO
	MinioUseSSL      *bool   `json:"minio_use_ssl"`     // Использовать SSL для MinIO
	fileObservabilityConfig
}

// PrintWorkerConfig записывает полный дамп настроек одной строкой в лог (вызывать после logger.Initialize).
func (c *ConfigWorker) PrintWorkerConfig() {
	var b strings.Builder

	fmt.Fprintf(&b, "LogLevel=%s RunMigrations=%t Config=%s; ", c.LogLevel, c.RunMigrations, c.Config)
	fmt.Fprintf(&b, "RabbitMQHost=%s RabbitMQPort=%d RabbitMQUser=%s RabbitMQPassword=%s RabbitMQVHost=%s; ",
		c.RabbitMQHost, c.RabbitMQPort, c.RabbitMQUser, redactSecret(c.RabbitMQPassword), c.RabbitMQVHost)
	fmt.Fprintf(&b, "MaxFileSize=%d; ", c.MaxFileSize)
	fmt.Fprintf(&b, "MinioEndpoint=%s MinioAccessKey=%s MinioSecretKey=%s MinioBucketName=%s MinioUseSSL=%t; ",
		c.MinioEndpoint, c.MinioAccessKey, redactSecret(c.MinioSecretKey), c.MinioBucketName, c.MinioUseSSL)
	fmt.Fprintf(&b, "OtelEnabled=%t OtelEndpoint=%s OtelServiceName=%s MetricsAddr=%s LogFormat=%s; ",
		c.Observability.OtelEnabled, c.Observability.OtelEndpoint, c.Observability.OtelServiceName, c.Observability.MetricsAddr, c.Observability.LogFormat)
	logger.Log.Info(b.String())
}

// redactSecret заменяет строку на "***"
func redactSecret(s string) string {
	if s == "" {
		return ""
	}
	return "***"
}

// InitConfigWorker возвращает настройки и ошибку если парсинг аргументов не удался.
// Порядок приоритета: значения по умолчанию → JSON-файл → переменные окружения → флаги.
func InitConfigWorker() (*ConfigWorker, error) {
	args := os.Args[1:]

	flagCfg, fs, err := parseWorkerFlags(args)
	if err != nil {
		return nil, err
	}

	configPath := strings.TrimSpace(flagCfg.Config)
	if !fs.Changed("config") {
		if v, ok := os.LookupEnv("CONFIG"); ok {
			configPath = strings.TrimSpace(v)
		}
	}

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

	if _, err := applyEnvToConfigWorker(&cfg, fs.Changed("config")); err != nil {
		return nil, err
	}

	applyExplicitWorkerFlags(&cfg, flagCfg, fs)

	var errs []error
	if cfg.RabbitMQPort < 1 || cfg.RabbitMQPort > 65535 {
		errs = append(errs, fmt.Errorf("rabbitmq_port: must be between 1 and 65535"))
	}
	if cfg.MaxFileSize < 1 {
		errs = append(errs, errors.New("max_file_size: must be positive"))
	}

	return &cfg, errors.Join(errs...)
}

// applyEnvToConfigWorker применяет переменные окружения. Если skipConfigFromEnv, CONFIG не трогаем
// (путь к файлу задан явно флагом -c).
func applyEnvToConfigWorker(config *ConfigWorker, skipConfigFromEnv bool) (*ConfigWorker, error) {
	if logLevel, present := os.LookupEnv("LOG_LEVEL"); present {
		config.LogLevel = logLevel
	}

	if databaseDSN, present := os.LookupEnv("DATABASE_DSN"); present {
		config.DatabaseDSN = databaseDSN
	}

	if runMigrations, present := os.LookupEnv("RUN_MIGRATIONS"); present {
		config.RunMigrations, _ = strconv.ParseBool(runMigrations)
	}

	if host, present := os.LookupEnv("RABBITMQ_HOST"); present {
		config.RabbitMQHost = host
	}

	if err := parseIntFromWorkerEnv(config, "RABBITMQ_PORT",
		func(c *ConfigWorker, v int) { c.RabbitMQPort = v }); err != nil {
		return nil, err
	}

	if user, present := os.LookupEnv("RABBITMQ_USER"); present {
		config.RabbitMQUser = user
	}

	if pass, present := os.LookupEnv("RABBITMQ_PASSWORD"); present {
		config.RabbitMQPassword = pass
	}

	if vhost, present := os.LookupEnv("RABBITMQ_VHOST"); present {
		config.RabbitMQVHost = vhost
	}

	if maxFileSize, present := os.LookupEnv("MAX_FILE_SIZE"); present {
		config.MaxFileSize, _ = strconv.Atoi(maxFileSize)
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

	applyObservabilityEnv(&config.Observability)

	return config, nil
}

// InitConfigWorkerWithArgs инициализация с переданными аргументами (только флаги; как раньше для тестов).
func InitConfigWorkerWithArgs(args []string) (*ConfigWorker, error) {
	cfg, _, err := parseWorkerFlags(args)
	return cfg, err
}

// parseWorkerFlags парсит флаги для воркера
func parseWorkerFlags(args []string) (*ConfigWorker, *flag.FlagSet, error) {
	var config ConfigWorker

	flagSet := flag.NewFlagSet("main", flag.ContinueOnError)

	flagSet.StringVarP(&config.LogLevel, "log-level", "l", "info", "log level")

	flagSet.StringVarP(&config.DatabaseDSN, "database-dsn", "d", getDefaultDatabaseDSN(), "database DSN")
	flagSet.BoolVarP(&config.RunMigrations, "run-migrations", "r", false, "run migrations")

	flagSet.StringVarP(&config.Config, "config", "c", getDefaultWorkerConfigFile(), "config file")

	flagSet.StringVar(&config.RabbitMQHost, "rabbitmq-host", getDefaultRabbitMQHost(), "RabbitMQ host")
	flagSet.IntVar(&config.RabbitMQPort, "rabbitmq-port", getDefaultRabbitMQPort(), "RabbitMQ AMQP port")
	flagSet.StringVar(&config.RabbitMQUser, "rabbitmq-user", getDefaultRabbitMQUser(), "RabbitMQ user")
	flagSet.StringVar(&config.RabbitMQPassword, "rabbitmq-password", getDefaultRabbitMQPassword(), "RabbitMQ password")
	flagSet.StringVar(&config.RabbitMQVHost, "rabbitmq-vhost", getDefaultRabbitMQVHost(), "RabbitMQ virtual host")

	flagSet.IntVar(&config.MaxFileSize, "max-file-size", getDefaultMaxFileSize(), "maximum input file size for processing (bytes)")

	flagSet.StringVar(&config.MinioEndpoint, "minio-endpoint", "", "MinIO endpoint")
	flagSet.StringVar(&config.MinioAccessKey, "minio-access-key", "", "MinIO access key")
	flagSet.StringVar(&config.MinioSecretKey, "minio-secret-key", "", "MinIO secret key")
	flagSet.StringVar(&config.MinioBucketName, "minio-bucket-name", "", "MinIO bucket name")
	flagSet.BoolVar(&config.MinioUseSSL, "minio-use-ssl", false, "Use SSL for MinIO")

	err := flagSet.Parse(args)
	if err != nil {
		return nil, nil, err
	}

	return &config, flagSet, nil
}

// defaultConfigWorker возвращает конфигурацию воркера по умолчанию
func defaultConfigWorker() ConfigWorker {
	return ConfigWorker{
		LogLevel:         "info",
		DatabaseDSN:      getDefaultDatabaseDSN(),
		RunMigrations:    false,
		Config:           "",
		RabbitMQHost:     getDefaultRabbitMQHost(),
		RabbitMQPort:     getDefaultRabbitMQPort(),
		RabbitMQUser:     getDefaultRabbitMQUser(),
		RabbitMQPassword: getDefaultRabbitMQPassword(),
		RabbitMQVHost:    getDefaultRabbitMQVHost(),
		MaxFileSize:      getDefaultMaxFileSize(),
		MinioEndpoint:    "",
		MinioAccessKey:   "",
		MinioSecretKey:   "",
		MinioBucketName:  "",
		MinioUseSSL:      false,
		Observability:    defaultObservabilityConfig(),
	}
}

// mergeConfigWorkerFromFile объединяет конфигурацию воркера с данными из файла
func mergeConfigWorkerFromFile(cfg *ConfigWorker, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var fc fileConfigWorker
	if err := json.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("invalid config file %s: %w", path, err)
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
	if fc.RabbitMQHost != nil {
		cfg.RabbitMQHost = *fc.RabbitMQHost
	}
	if fc.RabbitMQPort != nil {
		cfg.RabbitMQPort = *fc.RabbitMQPort
	}
	if fc.RabbitMQUser != nil {
		cfg.RabbitMQUser = *fc.RabbitMQUser
	}
	if fc.RabbitMQPassword != nil {
		cfg.RabbitMQPassword = *fc.RabbitMQPassword
	}
	if fc.RabbitMQVHost != nil {
		cfg.RabbitMQVHost = *fc.RabbitMQVHost
	}
	if fc.MaxFileSize != nil {
		cfg.MaxFileSize = *fc.MaxFileSize
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

	mergeObservabilityFromFile(&cfg.Observability, &fc.fileObservabilityConfig)

	return nil
}

// applyExplicitWorkerFlags применяет переданные флаги к конфигурации воркера
func applyExplicitWorkerFlags(dst *ConfigWorker, src *ConfigWorker, fs *flag.FlagSet) {
	if fs.Changed("log-level") {
		dst.LogLevel = src.LogLevel
	}
	if fs.Changed("database-dsn") {
		dst.DatabaseDSN = src.DatabaseDSN
	}
	if fs.Changed("run-migrations") {
		dst.RunMigrations = src.RunMigrations
	}
	if fs.Changed("config") {
		dst.Config = strings.TrimSpace(src.Config)
	}
	if fs.Changed("rabbitmq-host") {
		dst.RabbitMQHost = src.RabbitMQHost
	}
	if fs.Changed("rabbitmq-port") {
		dst.RabbitMQPort = src.RabbitMQPort
	}
	if fs.Changed("rabbitmq-user") {
		dst.RabbitMQUser = src.RabbitMQUser
	}
	if fs.Changed("rabbitmq-password") {
		dst.RabbitMQPassword = src.RabbitMQPassword
	}
	if fs.Changed("rabbitmq-vhost") {
		dst.RabbitMQVHost = src.RabbitMQVHost
	}
	if fs.Changed("max-file-size") {
		dst.MaxFileSize = src.MaxFileSize
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

func getDefaultWorkerConfigFile() string {
	return "config_worker.json"
}

func getDefaultRabbitMQHost() string {
	return "127.0.0.1"
}

func getDefaultRabbitMQPort() int {
	return 5672
}

func getDefaultRabbitMQUser() string {
	return "gophprofile"
}

func getDefaultRabbitMQPassword() string {
	return "gophprofile"
}

func getDefaultRabbitMQVHost() string {
	return "/"
}
