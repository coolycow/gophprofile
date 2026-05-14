package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadFromFlagsAndEnv парсит флаги и затем применяет переменные окружения (как в ручных сценариях тестирования).
func loadFromFlagsAndEnv(t *testing.T, args []string) *ConfigServer {
	t.Helper()
	cfg, err := InitConfigServerWithArgs(args)
	require.NoError(t, err)
	cfg, err = initConfigServerWithEnv(cfg)
	require.NoError(t, err)
	return cfg
}

func TestInitConfigServerWithArgs_defaults(t *testing.T) {
	cfg, err := InitConfigServerWithArgs(nil)
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "", cfg.DatabaseDSN)
	assert.False(t, cfg.RunMigrations)
	assert.False(t, cfg.EnableHTTPS)
	assert.Equal(t, getDefaultTLSCertFile(), cfg.TLSCertFile)
	assert.Equal(t, getDefaultTLSKeyFile(), cfg.TLSKeyFile)
	assert.Equal(t, getDefaultConfigFile(), cfg.Config)
	assert.Empty(t, cfg.TrustedSubnet)
	assert.Equal(t, getDefaultSecretKey(), cfg.SecretKey)
	assert.Equal(t, getDefaultMinPasswordLength(), cfg.MinPasswordLength)
	assert.Equal(t, getDefaultMaxPasswordLength(), cfg.MaxPasswordLength)
	assert.Empty(t, cfg.AuditFile)
	assert.Empty(t, cfg.AuditURL)
	assert.Equal(t, getDefaultAccessTokenTTLMinutes(), cfg.AccessTokenTTLMinutes)
	assert.Equal(t, getDefaultRefreshTokenTTLHours(), cfg.RefreshTokenTTLHours)
}

func TestInitConfigServerWithArgs_flags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want func(*testing.T, *ConfigServer)
	}{
		{
			name: "host and port",
			args: []string{"--host", "localhost", "--port", "8082"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "localhost", c.Host)
				assert.Equal(t, 8082, c.Port)
			},
		},
		{
			name: "log level",
			args: []string{"--log-level", "debug"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "debug", c.LogLevel)
			},
		},
		{
			name: "database dsn",
			args: []string{"--database-dsn", "postgres://localhost/db"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "postgres://localhost/db", c.DatabaseDSN)
			},
		},
		{
			name: "run migrations",
			args: []string{"--run-migrations", "true"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.True(t, c.RunMigrations)
			},
		},
		{
			name: "enable https",
			args: []string{"--enable-https", "true"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.True(t, c.EnableHTTPS)
			},
		},
		{
			name: "tls files",
			args: []string{"--tls-cert-file", "cert.pem", "--tls-key-file", "key.pem"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "cert.pem", c.TLSCertFile)
				assert.Equal(t, "key.pem", c.TLSKeyFile)
			},
		},
		{
			name: "config path",
			args: []string{"--config", "custom.json"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "custom.json", c.Config)
			},
		},
		{
			name: "trusted subnet",
			args: []string{"--trusted-subnet", "192.168.0.0/16"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "192.168.0.0/16", c.TrustedSubnet)
			},
		},
		{
			name: "secret key",
			args: []string{"--secret-key", "01234567890123456789012345678901"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "01234567890123456789012345678901", c.SecretKey)
			},
		},
		{
			name: "password length bounds",
			args: []string{"--min-password-length", "8", "--max-password-length", "64"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, 8, c.MinPasswordLength)
				assert.Equal(t, 64, c.MaxPasswordLength)
			},
		},
		{
			name: "audit file and url",
			args: []string{"--audit-file", "audit.log", "--audit-url", "http://localhost/audit"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "audit.log", c.AuditFile)
				assert.Equal(t, "http://localhost/audit", c.AuditURL)
			},
		},
		{
			name: "jwt and refresh ttl",
			args: []string{"--access-token-ttl-min", "30", "--refresh-token-ttl-hours", "720"},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, 30, c.AccessTokenTTLMinutes)
				assert.Equal(t, 720, c.RefreshTokenTTLHours)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := InitConfigServerWithArgs(tt.args)
			require.NoError(t, err)
			tt.want(t, cfg)
		})
	}
}

func TestInitConfigServerWithEnv_supportedVars(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want func(*testing.T, *ConfigServer)
	}{
		{
			name: "host port grpc",
			env: map[string]string{
				"HOST":      "10.0.0.1",
				"PORT":      "9000",
				"GRPC_PORT": "50052",
			},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "10.0.0.1", c.Host)
				assert.Equal(t, 9000, c.Port)
			},
		},
		{
			name: "log and database",
			env: map[string]string{
				"LOG_LEVEL":      "warn",
				"DATABASE_DSN":   "postgres://x",
				"RUN_MIGRATIONS": "true",
			},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "warn", c.LogLevel)
				assert.Equal(t, "postgres://x", c.DatabaseDSN)
				assert.True(t, c.RunMigrations)
			},
		},
		{
			name: "https and tls",
			env: map[string]string{
				"ENABLE_HTTPS":  "true",
				"TLS_CERT_FILE": "c.pem",
				"TLS_KEY_FILE":  "k.pem",
			},
			want: func(t *testing.T, c *ConfigServer) {
				assert.True(t, c.EnableHTTPS)
				assert.Equal(t, "c.pem", c.TLSCertFile)
				assert.Equal(t, "k.pem", c.TLSKeyFile)
			},
		},
		{
			name: "secret and crypto params",
			env: map[string]string{
				"SECRET_KEY":           "01234567890123456789012345678901",
				"SECRET_VERSION_COUNT": "5",
				"SALT_LENGTH":          "24",
				"MIN_PASSWORD_LENGTH":  "10",
				"MAX_PASSWORD_LENGTH":  "128",
			},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "01234567890123456789012345678901", c.SecretKey)
				assert.Equal(t, 10, c.MinPasswordLength)
				assert.Equal(t, 128, c.MaxPasswordLength)
			},
		},
		{
			name: "audit and subnet",
			env: map[string]string{
				"AUDIT_FILE":     "audit.json",
				"AUDIT_URL":      "http://audit",
				"TRUSTED_SUBNET": "10.0.0.0/8",
			},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "audit.json", c.AuditFile)
				assert.Equal(t, "http://audit", c.AuditURL)
				assert.Equal(t, "10.0.0.0/8", c.TrustedSubnet)
			},
		},
		{
			name: "token ttl",
			env: map[string]string{
				"ACCESS_TOKEN_TTL_MIN": "60",
				"REFRESH_TOKEN_TTL_H":  "24",
			},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, 60, c.AccessTokenTTLMinutes)
				assert.Equal(t, 24, c.RefreshTokenTTLHours)
			},
		},
		{
			name: "config path",
			env: map[string]string{
				"CONFIG": "from-env.json",
			},
			want: func(t *testing.T, c *ConfigServer) {
				assert.Equal(t, "from-env.json", c.Config)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			cfg := loadFromFlagsAndEnv(t, nil)
			tt.want(t, cfg)
		})
	}
}

func TestEnvOverridesFlags(t *testing.T) {
	t.Setenv("HOST", "from-env")
	t.Setenv("PORT", "7777")

	cfg := loadFromFlagsAndEnv(t, []string{"--host", "from-flag", "--port", "6666"})
	assert.Equal(t, "from-env", cfg.Host)
	assert.Equal(t, 7777, cfg.Port)
}

func TestInitConfigServer_invalidTrustedSubnet(t *testing.T) {
	t.Setenv("TRUSTED_SUBNET", "not-a-cidr")
	_, err := InitConfigServer()
	require.Error(t, err)
}
