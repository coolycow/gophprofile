package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/coolycow/gophprofile/internal/logger"
)

type ConfigServer struct {
	Host                  string `env:"HOST" json:"host,omitempty"`                                     // IP адрес сервера
	Port                  int    `env:"PORT" json:"port,omitempty"`                                     // Порт сервера
	LogLevel              string `env:"LOG_LEVEL" json:"log_level,omitempty"`                           // Уровень логирования
	DatabaseDSN           string `env:"DATABASE_DSN" json:"database_dsn,omitempty"`                     // DSN для подключения к БД
	RunMigrations         bool   `env:"RUN_MIGRATIONS" json:"run_migrations,omitempty"`                 // Флаг для запуска миграций
	EnableHTTPS           bool   `env:"ENABLE_HTTPS" json:"enable_https,omitempty"`                     // Флаг для включения HTTPS
	TLSCertFile           string `env:"TLS_CERT_FILE" json:"tls_cert_file,omitempty"`                   // Файл сертификата для HTTPS
	TLSKeyFile            string `env:"TLS_KEY_FILE" json:"tls_key_file,omitempty"`                     // Файл ключа для HTTPS
	TrustedSubnet         string `env:"TRUSTED_SUBNET" json:"trusted_subnet,omitempty"`                 // Подсеть для доступа к статистике
	Config                string `env:"CONFIG" json:"config,omitempty"`                                 // Путь к файлу конфигурации
	SecretKey             string `env:"SECRET_KEY" json:"secret_key,omitempty"`                         // Секретный ключ для шифрования данных
	MinPasswordLength     int    `env:"MIN_PASSWORD_LENGTH" json:"min_password_length,omitempty"`       // Минимальная длина пароля
	MaxPasswordLength     int    `env:"MAX_PASSWORD_LENGTH" json:"max_password_length,omitempty"`       // Максимальная длина пароля
	AuditFile             string `env:"AUDIT_FILE" json:"audit_file,omitempty"`                         // Путь к файлу аудита
	AuditURL              string `env:"AUDIT_URL" json:"audit_url,omitempty"`                           // URL для отправки аудита
	AccessTokenTTLMinutes int    `env:"ACCESS_TOKEN_TTL_MIN" json:"access_token_ttl_minutes,omitempty"` // Срок жизни JWT access (минуты)
	RefreshTokenTTLHours  int    `env:"REFRESH_TOKEN_TTL_H" json:"refresh_token_ttl_hours,omitempty"`   // Срок жизни refresh в БД (часы)
	MaxFileSize           int    `env:"MAX_FILE_SIZE" json:"max_file_size,omitempty"`                   // Максимальный размер файла (байты)

	MinioEndpoint   string `env:"MINIO_ENDPOINT" json:"minio_endpoint,omitempty"`       // Endpoint MinIO
	MinioAccessKey  string `env:"MINIO_ACCESS_KEY" json:"minio_access_key,omitempty"`   // Access Key MinIO
	MinioSecretKey  string `env:"MINIO_SECRET_KEY" json:"minio_secret_key,omitempty"`   // Secret Key MinIO
	MinioBucketName string `env:"MINIO_BUCKET_NAME" json:"minio_bucket_name,omitempty"` // Имя бакета MinIO
	MinioUseSSL     bool   `env:"MINIO_USE_SSL" json:"minio_use_ssl,omitempty"`         // Использовать SSL для MinIO
}

// fileConfig — JSON-файл; указатели задают поля, явно присутствующие в файле.
// Ключ "config" в файле не разбираем (путь к файлу только из -c / CONFIG).
type fileConfigServer struct {
	Host                  *string `json:"host"`                     // IP адрес сервера
	Port                  *int    `json:"port"`                     // Порт сервера
	LogLevel              *string `json:"log_level"`                // Уровень логирования
	DatabaseDSN           *string `json:"database_dsn"`             // DSN для подключения к БД
	RunMigrations         *bool   `json:"run_migrations"`           // Флаг для запуска миграций
	EnableHTTPS           *bool   `json:"enable_https"`             // Флаг для включения HTTPS
	TLSCertFile           *string `json:"tls_cert_file"`            // Файл сертификата для HTTPS
	TLSKeyFile            *string `json:"tls_key_file"`             // Файл ключа для HTTPS
	TrustedSubnet         *string `json:"trusted_subnet"`           // Подсеть для доступа к статистике
	SecretKey             *string `json:"secret_key"`               // Секретный ключ для шифрования данных
	MinPasswordLength     *int    `json:"min_password_length"`      // Минимальная длина пароля
	MaxPasswordLength     *int    `json:"max_password_length"`      // Максимальная длина пароля
	AuditFile             *string `json:"audit_file"`               // Путь к файлу аудита
	AuditURL              *string `json:"audit_url"`                // URL для отправки аудита
	AccessTokenTTLMinutes *int    `json:"access_token_ttl_minutes"` // JWT access TTL (минуты)
	RefreshTokenTTLHours  *int    `json:"refresh_token_ttl_hours"`  // refresh TTL (часы)
	MaxFileSize           *int    `json:"max_file_size"`            // Максимальный размер файла (байты)
	MinioEndpoint         *string `json:"minio_endpoint"`           // Endpoint MinIO
	MinioAccessKey        *string `json:"minio_access_key"`         // Access Key MinIO
	MinioSecretKey        *string `json:"minio_secret_key"`         // Secret Key MinIO
	MinioBucketName       *string `json:"minio_bucket_name"`        // Имя бакета MinIO
	MinioUseSSL           *bool   `json:"minio_use_ssl"`            // Использовать SSL для MinIO
}

// GetServerAddress возвращает полный адрес сервера для его запуска
func (c *ConfigServer) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// PrintServerConfig записывает полный дамп настроек одной строкой в лог (вызывать после logger.Initialize).
func (c *ConfigServer) PrintServerConfig() {
	var b strings.Builder

	// Формируем строку с настройками
	fmt.Fprintf(&b, "Host=%s Port=%d ", c.Host, c.Port)
	fmt.Fprintf(&b, "LogLevel=%s RunMigrations=%t; ", c.LogLevel, c.RunMigrations)
	fmt.Fprintf(&b, "EnableHTTPS=%t TLSCertFile=%s TLSKeyFile=%s TrustedSubnet=%s Config=%s; ",
		c.EnableHTTPS, c.TLSCertFile, c.TLSKeyFile, c.TrustedSubnet, c.Config)
	fmt.Fprintf(&b, "MinPasswordLength=%d MaxPasswordLength=%d; ", c.MinPasswordLength, c.MaxPasswordLength)
	fmt.Fprintf(&b, "AccessTokenTTLMinutes=%d RefreshTokenTTLHours=%d; ", c.AccessTokenTTLMinutes, c.RefreshTokenTTLHours)
	fmt.Fprintf(&b, "AuditFile=%s AuditURL=%s; ", c.AuditFile, c.AuditURL)
	fmt.Fprintf(&b, "MaxFileSize=%d; ", c.MaxFileSize)
	fmt.Fprintf(&b, "MinioEndpoint=%s MinioAccessKey=%s MinioSecretKey=%s MinioBucketName=%s MinioUseSSL=%t; ", c.MinioEndpoint, c.MinioAccessKey, c.MinioSecretKey, c.MinioBucketName, c.MinioUseSSL)
	// Выводим настройки в лог
	logger.Log.Info(b.String())
}

// InitConfigServer возвращает настройки и ошибку если парсинг аргументов не удался.
// Порядок приоритета: значения по умолчанию → JSON-файл → переменные окружения → флаги.
func InitConfigServer() (*ConfigServer, error) {
	args := os.Args[1:]

	// Получаем настройки из флагов
	flagCfg, fs, err := parseServerFlags(args)
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
	cfg := defaultConfigServer()
	_, configEnvSet := os.LookupEnv("CONFIG")
	explicitConfigPath := fs.Changed("config") || configEnvSet

	if configPath != "" {
		if err := mergeConfigServerFromFile(&cfg, configPath); err != nil {
			if errors.Is(err, os.ErrNotExist) && !explicitConfigPath {
				cfg = defaultConfigServer()
			} else {
				return nil, err
			}
		}
	}

	cfg.Config = configPath

	// Применяем настройки из переменных окружения
	if _, err := applyEnvToConfigServer(&cfg, fs.Changed("config")); err != nil {
		return nil, err
	}

	applyExplicitServerFlags(&cfg, flagCfg, fs)

	// Проверяем настройки на корректность
	var errs []error
	if ts := strings.TrimSpace(cfg.TrustedSubnet); ts != "" {
		if _, _, err := net.ParseCIDR(ts); err != nil {
			errs = append(errs, fmt.Errorf("trusted_subnet: %w", err))
		}
	}

	if cfg.SecretKey == "" {
		errs = append(errs, errors.New("secret key must be set"))
	}

	if len(cfg.SecretKey) != 32 {
		errs = append(errs, errors.New("secret key must be 32 characters long"))
	}

	if cfg.AccessTokenTTLMinutes < 1 || cfg.AccessTokenTTLMinutes > 1440 {
		errs = append(errs, errors.New("access_token_ttl_minutes must be between 1 and 1440"))
	}
	if cfg.RefreshTokenTTLHours < 1 || cfg.RefreshTokenTTLHours > 8760 {
		errs = append(errs, errors.New("refresh_token_ttl_hours must be between 1 and 8760"))
	}

	return &cfg, errors.Join(errs...)
}

// initConfigServerWithEnv получение настроек из переменных окружения.
func initConfigServerWithEnv(config *ConfigServer) (*ConfigServer, error) {
	return applyEnvToConfigServer(config, false)
}

// applyEnvToConfigServer применяет переменные окружения. Если skipConfigFromEnv, CONFIG не трогаем
// (путь к файлу задан явно флагом -c).
func applyEnvToConfigServer(config *ConfigServer, skipConfigFromEnv bool) (*ConfigServer, error) {
	if host, present := os.LookupEnv("HOST"); present {
		config.Host = host
	}

	if err := parseIntFromServerEnv(config, "PORT",
		func(c *ConfigServer, v int) { c.Port = v }); err != nil {
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

	if secretKey, present := os.LookupEnv("SECRET_KEY"); present {
		config.SecretKey = secretKey
	}

	if minPasswordLength, present := os.LookupEnv("MIN_PASSWORD_LENGTH"); present {
		config.MinPasswordLength, _ = strconv.Atoi(minPasswordLength)
	}

	if maxPasswordLength, present := os.LookupEnv("MAX_PASSWORD_LENGTH"); present {
		config.MaxPasswordLength, _ = strconv.Atoi(maxPasswordLength)
	}

	if auditFile, present := os.LookupEnv("AUDIT_FILE"); present {
		config.AuditFile = auditFile
	}

	if auditURL, present := os.LookupEnv("AUDIT_URL"); present {
		config.AuditURL = auditURL
	}

	if err := parseIntFromServerEnv(config, "ACCESS_TOKEN_TTL_MIN",
		func(c *ConfigServer, v int) { c.AccessTokenTTLMinutes = v }); err != nil {
		return nil, err
	}
	if err := parseIntFromServerEnv(config, "REFRESH_TOKEN_TTL_H",
		func(c *ConfigServer, v int) { c.RefreshTokenTTLHours = v }); err != nil {
		return nil, err
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

	return config, nil
}

// InitConfigServerWithArgs инициализация с переданными аргументами (только флаги; как раньше для тестов).
func InitConfigServerWithArgs(args []string) (*ConfigServer, error) {
	cfg, _, err := parseServerFlags(args)
	return cfg, err
}

// parseServerFlags парсит флаги для сервера
func parseServerFlags(args []string) (*ConfigServer, *flag.FlagSet, error) {
	var config ConfigServer

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

	// Флаги для секретного ключа
	flagSet.StringVarP(&config.SecretKey, "secret-key", "x", getDefaultSecretKey(), "secret key")

	// Флаги для длины пароля
	flagSet.IntVarP(&config.MinPasswordLength, "min-password-length", "o", getDefaultMinPasswordLength(), "minimum password length")
	// без «p»: занят портом (--port)
	flagSet.IntVar(&config.MaxPasswordLength, "max-password-length", getDefaultMaxPasswordLength(), "maximum password length")

	// Флаги для аудита
	flagSet.StringVarP(&config.AuditFile, "audit-file", "a", "", "audit file")
	flagSet.StringVarP(&config.AuditURL, "audit-url", "b", "", "audit URL")

	// JWT access и refresh в БД
	flagSet.IntVar(&config.AccessTokenTTLMinutes, "access-token-ttl-min", getDefaultAccessTokenTTLMinutes(), "JWT access token TTL in minutes")
	flagSet.IntVar(&config.RefreshTokenTTLHours, "refresh-token-ttl-hours", getDefaultRefreshTokenTTLHours(), "opaque refresh token TTL stored in DB (hours)")

	// Флаги для максимального размера файла
	flagSet.IntVar(&config.MaxFileSize, "max-file-size", getDefaultMaxFileSize(), "maximum file size (bytes)")

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

// defaultConfigServer возвращает конфигурацию сервера по умолчанию
func defaultConfigServer() ConfigServer {
	return ConfigServer{
		Host:                  "127.0.0.1",
		Port:                  8080,
		LogLevel:              "info",
		DatabaseDSN:           getDefaultDatabaseDSN(),
		RunMigrations:         false,
		EnableHTTPS:           false,
		TLSCertFile:           getDefaultTLSCertFile(),
		TLSKeyFile:            getDefaultTLSKeyFile(),
		TrustedSubnet:         "",
		Config:                "",
		SecretKey:             getDefaultSecretKey(),
		MinPasswordLength:     getDefaultMinPasswordLength(),
		MaxPasswordLength:     getDefaultMaxPasswordLength(),
		AuditFile:             getDefaultAuditFile(),
		AuditURL:              getDefaultAuditURL(),
		AccessTokenTTLMinutes: getDefaultAccessTokenTTLMinutes(),
		RefreshTokenTTLHours:  getDefaultRefreshTokenTTLHours(),
		MaxFileSize:           getDefaultMaxFileSize(),
		MinioEndpoint:         "",
		MinioAccessKey:        "",
		MinioSecretKey:        "",
		MinioBucketName:       "",
		MinioUseSSL:           false,
	}
}

// mergeConfigServerFromFile объединяет конфигурацию сервера с данными из файла
func mergeConfigServerFromFile(cfg *ConfigServer, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var fc fileConfigServer
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
	if fc.SecretKey != nil {
		cfg.SecretKey = *fc.SecretKey
	}

	if fc.MinPasswordLength != nil {
		cfg.MinPasswordLength = *fc.MinPasswordLength
	}
	if fc.MaxPasswordLength != nil {
		cfg.MaxPasswordLength = *fc.MaxPasswordLength
	}

	if fc.AuditFile != nil {
		cfg.AuditFile = *fc.AuditFile
	}
	if fc.AuditURL != nil {
		cfg.AuditURL = *fc.AuditURL
	}
	if fc.AccessTokenTTLMinutes != nil {
		cfg.AccessTokenTTLMinutes = *fc.AccessTokenTTLMinutes
	}
	if fc.RefreshTokenTTLHours != nil {
		cfg.RefreshTokenTTLHours = *fc.RefreshTokenTTLHours
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
	return nil
}

// applyExplicitServerFlags применяет переданные флаги к конфигурации сервера
func applyExplicitServerFlags(dst *ConfigServer, src *ConfigServer, fs *flag.FlagSet) {
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
	if fs.Changed("secret-key") {
		dst.SecretKey = src.SecretKey
	}
	if fs.Changed("min-password-length") {
		dst.MinPasswordLength = src.MinPasswordLength
	}
	if fs.Changed("max-password-length") {
		dst.MaxPasswordLength = src.MaxPasswordLength
	}
	if fs.Changed("audit-file") {
		dst.AuditFile = src.AuditFile
	}
	if fs.Changed("audit-url") {
		dst.AuditURL = src.AuditURL
	}
	if fs.Changed("access-token-ttl-min") {
		dst.AccessTokenTTLMinutes = src.AccessTokenTTLMinutes
	}
	if fs.Changed("refresh-token-ttl-hours") {
		dst.RefreshTokenTTLHours = src.RefreshTokenTTLHours
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

// parseIntFromServerEnv парсит int-значение из переменной окружения и устанавливает его в поле конфигурации
func parseIntFromServerEnv(config *ConfigServer, envKey string, setter func(*ConfigServer, int)) error {
	if value, present := os.LookupEnv(envKey); present {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid env %s %s", envKey, value)
		}
		setter(config, intValue)
	}
	return nil
}

// getDefaultDatabaseDSN Стандартные настройки подключения к БД
func getDefaultDatabaseDSN() string {
	return ""
}

// getDefaultSecretKey секретный ключ по умолчанию
func getDefaultSecretKey() string {
	return "phWHXBnGVl4JyzNbYRL3MSRL07pir7i6"
}

// getDefaultTLSCertFile файл сертификата по умолчанию
func getDefaultTLSCertFile() string {
	return "server.crt"
}

// getDefaultTLSKeyFile файл ключа по умолчанию
func getDefaultTLSKeyFile() string {
	return "server.key"
}

// getDefaultConfigFile файл конфигурации по умолчанию
func getDefaultConfigFile() string {
	return "config_server.json"
}

// getDefaultMinPasswordLength минимальная длина пароля
func getDefaultMinPasswordLength() int {
	return 3
}

// getDefaultMaxPasswordLength максимальная длина пароля
func getDefaultMaxPasswordLength() int {
	return 255
}

// getDefaultAuditFile файл аудита по умолчанию (т.к. задаётся реальное название, то аудит будет записываться в файл если принудительно не передать пустое значение)
func getDefaultAuditFile() string {
	return "audit.json"
}

// getDefaultAuditURL URL для отправки аудита по умолчанию
func getDefaultAuditURL() string {
	return ""
}

// getDefaultAccessTokenTTLMinutes срок жизни JWT access по умолчанию.
func getDefaultAccessTokenTTLMinutes() int {
	return 15
}

// getDefaultRefreshTokenTTLHours срок хранения refresh в БД по умолчанию.
func getDefaultRefreshTokenTTLHours() int {
	return 168
}

// getDefaultMaxFileSize максимальный размер файла по умолчанию (10MB в байтах)
func getDefaultMaxFileSize() int {
	return 1024000
}
