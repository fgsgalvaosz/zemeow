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

type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	Auth     AuthConfig
	WhatsApp WhatsAppConfig
	Logging  LoggingConfig
	Webhook  WebhookConfig
	MinIO    MinIOConfig
}

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

type ServerConfig struct {
	Host         string
	Port         int
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type AuthConfig struct {
	AdminAPIKey string
}

type WhatsAppConfig struct {
	Timeout           time.Duration
	ReconnectInterval time.Duration
	QRCodeTimeout     time.Duration
}

type LoggingConfig struct {
	Level  string
	Pretty bool
}

type WebhookConfig struct {
	Timeout       time.Duration
	RetryCount    int
	RetryInterval time.Duration
}

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	UseSSL          bool
	Region          string
	PublicURL       string
}

func Load() (*Config, error) {

	if err := godotenv.Load(); err != nil {

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
		MinIO: MinIOConfig{
			Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKeyID:     getEnv("MINIO_ACCESS_KEY", "Gacont"),
			SecretAccessKey: getEnv("MINIO_SECRET_KEY", "WIPcLhjcBoslmOd"),
			BucketName:      getEnv("MINIO_BUCKET_NAME", "zemeow-media"),
			UseSSL:          getEnvAsBool("MINIO_USE_SSL", false),
			Region:          getEnv("MINIO_REGION", "us-east-1"),
			PublicURL:       getEnv("MINIO_PUBLIC_URL", ""),
		},
	}

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

func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

func (c *Config) GetServerAddress() string {
	return net.JoinHostPort(c.Server.Host, fmt.Sprintf("%d", c.Server.Port))
}

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

		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}

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
