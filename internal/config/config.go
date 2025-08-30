package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config contém todas as configurações da aplicação
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	Auth     AuthConfig
	WhatsApp WhatsAppConfig
	Logging  LoggingConfig
	Webhook  WebhookConfig
}

// DatabaseConfig configurações do banco de dados
type DatabaseConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// ServerConfig configurações do servidor HTTP
type ServerConfig struct {
	Host         string
	Port         int
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// AuthConfig configurações de autenticação
type AuthConfig struct {
	AdminAPIKey string
}

// WhatsAppConfig configurações do WhatsApp
type WhatsAppConfig struct {
	Timeout           time.Duration
	ReconnectInterval time.Duration
	QRCodeTimeout     time.Duration
}

// LoggingConfig configurações de logging
type LoggingConfig struct {
	Level  string
	Pretty bool
}

// WebhookConfig configurações de webhook
type WebhookConfig struct {
	Timeout       time.Duration
	RetryCount    int
	RetryInterval time.Duration
}

// Load carrega as configurações das variáveis de ambiente
func Load() (*Config, error) {
	// Tentar carregar .env se existir
	if err := godotenv.Load(); err != nil {
		// Não é um erro crítico se .env não existir
		fmt.Printf("Warning: .env file not found: %v\n", err)
	}

	config := &Config{
		Database: DatabaseConfig{
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnvAsInt("POSTGRES_PORT", 5432),
			Name:            getEnv("POSTGRES_DB", "zemeow"),
			User:            getEnv("POSTGRES_USER", "zemeow"),
			Password:        getEnv("POSTGRES_PASSWORD", "zemeow123"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			URL:             getEnv("DATABASE_URL", ""),
			MaxOpenConns:    getEnvAsInt("DATABASE_MAX_OPEN_CONNS", 50),
			MaxIdleConns:    getEnvAsInt("DATABASE_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsDuration("DATABASE_CONN_MAX_LIFETIME", 15*time.Minute),
			ConnMaxIdleTime: getEnvAsDuration("DATABASE_CONN_MAX_IDLE_TIME", 5*time.Minute),
		},
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvAsInt("SERVER_PORT", 8080),
			Environment:  getEnv("ENVIRONMENT", "development"),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
		Auth: AuthConfig{
			AdminAPIKey: getEnv("ADMIN_API_KEY", "admin_secret_key"),
		},
		WhatsApp: WhatsAppConfig{
			Timeout:           getEnvAsDuration("WHATSAPP_TIMEOUT", 30*time.Second),
			ReconnectInterval: getEnvAsDuration("WHATSAPP_RECONNECT_INTERVAL", 5*time.Second),
			QRCodeTimeout:     getEnvAsDuration("QR_CODE_TIMEOUT", 60*time.Second),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Pretty: getEnvAsBool("LOG_PRETTY", true),
		},
		Webhook: WebhookConfig{
			Timeout:       getEnvAsDuration("WEBHOOK_TIMEOUT", 10*time.Second),
			RetryCount:    getEnvAsInt("WEBHOOK_RETRY_COUNT", 3),
			RetryInterval: getEnvAsDuration("WEBHOOK_RETRY_INTERVAL", 5*time.Second),
		},
	}

	// Construir URL do banco se não fornecida
	if config.Database.URL == "" {
		config.Database.URL = fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			config.Database.User,
			config.Database.Password,
			config.Database.Host,
			config.Database.Port,
			config.Database.Name,
			config.Database.SSLMode,
		)
	}

	return config, nil
}

// Validate valida as configurações
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Auth.AdminAPIKey == "" {
		return fmt.Errorf("admin API key is required")
	}
	return nil
}

// IsDevelopment retorna true se estiver em ambiente de desenvolvimento
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction retorna true se estiver em ambiente de produção
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// GetServerAddress retorna o endereço completo do servidor
func (c *Config) GetServerAddress() string {
	return net.JoinHostPort(c.Server.Host, fmt.Sprintf("%d", c.Server.Port))
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		// Tentar parsear como duração (ex: "30s", "5m")
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		// Tentar parsear como segundos
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string, separator string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, separator)
	}
	return defaultValue
}
