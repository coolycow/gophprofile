package config

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
}

// fileConfig — JSON-файл; указатели задают поля, явно присутствующие в файле.
// Ключ "config" в файле не разбираем (путь к файлу только из -c / CONFIG).
type fileConfigWorker struct {
	Host          *string `json:"host"`           // IP адрес сервера
	Port          *int    `json:"port"`           // Порт сервера
	LogLevel      *string `json:"log_level"`      // Уровень логирования
	DatabaseDSN   *string `json:"database_dsn"`   // DSN для подключения к БД
	RunMigrations *bool   `json:"run_migrations"` // Флаг для запуска миграций
	EnableHTTPS   *bool   `json:"enable_https"`   // Флаг для включения HTTPS
	TLSCertFile   *string `json:"tls_cert_file"`  // Файл сертификата для HTTPS
	TLSKeyFile    *string `json:"tls_key_file"`   // Файл ключа для HTTPS
	TrustedSubnet *string `json:"trusted_subnet"` // Подсеть для доступа к статистике
	Config        *string `json:"config"`         // Путь к файлу конфигурации
}
