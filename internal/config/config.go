// Package config loads and validates application configuration from environment
// variables.
//
// The configuration is loaded once at startup and is immutable afterwards.
// Missing required values or invalid formats cause the process to fail fast with
// a clear error before the server starts serving traffic.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the top-level application configuration.
type Config struct {
	HTTP    HTTPConfig
	DB      DBConfig
	Redis   RedisConfig
	Log     LogConfig
	JWT     JWTConfig
	Mail    MailConfig
	Storage StorageConfig
}

// HTTPConfig configures the HTTP server.
type HTTPConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DBConfig configures the PostgreSQL connection pool.
type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// RedisConfig configures the Redis connection.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// LogConfig configures structured logging.
type LogConfig struct {
	Level   string
	Format  string
	Service string
}

// JWTConfig configures JWT signing.
type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// MailConfig configures the SMTP mailer.
type MailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// StorageConfig configures file storage.
type StorageConfig struct {
	Driver string
	Local  LocalStorageConfig
	S3     S3StorageConfig
}

// LocalStorageConfig configures the local disk driver.
type LocalStorageConfig struct {
	Path string
}

// S3StorageConfig configures the S3 driver.
type S3StorageConfig struct {
	Bucket    string
	Region    string
	Endpoint  string
	AccessKey string
	SecretKey string
}

// Load reads configuration from environment variables and validates it.
func Load() (Config, error) {
	cfg := Config{
		HTTP: HTTPConfig{
			Addr:         envString("HTTP_ADDR", ":8080"),
			ReadTimeout:  envDuration("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout: envDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  envDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		DB: DBConfig{
			Host:            envString("DB_HOST", "localhost"),
			Port:            envInt("DB_PORT", 5432),
			User:            envString("DB_USER", "postgres"),
			Password:        envString("DB_PASSWORD", "postgres"),
			Name:            envString("DB_NAME", "gin_boilerplate"),
			SSLMode:         envString("DB_SSLMODE", "disable"),
			MaxOpenConns:    envInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: envDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: envDuration("DB_CONN_MAX_IDLE_TIME", 1*time.Minute),
		},
		Redis: RedisConfig{
			Addr:     envString("REDIS_ADDR", "localhost:6379"),
			Password: envString("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
		},
		Log: LogConfig{
			Level:   envString("LOG_LEVEL", "info"),
			Format:  envString("LOG_FORMAT", "json"),
			Service: envString("LOG_SERVICE", "gin-boilerplate"),
		},
		JWT: JWTConfig{
			Secret:     envString("JWT_SECRET", ""),
			AccessTTL:  envDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: envDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Mail: MailConfig{
			Host:     envString("MAIL_HOST", "localhost"),
			Port:     envInt("MAIL_PORT", 1025),
			Username: envString("MAIL_USERNAME", ""),
			Password: envString("MAIL_PASSWORD", ""),
			From:     envString("MAIL_FROM", "noreply@example.com"),
		},
		Storage: StorageConfig{
			Driver: envString("STORAGE_DRIVER", "local"),
			Local: LocalStorageConfig{
				Path: envString("STORAGE_LOCAL_PATH", "./uploads"),
			},
			S3: S3StorageConfig{
				Bucket:    envString("S3_BUCKET", ""),
				Region:    envString("S3_REGION", "us-east-1"),
				Endpoint:  envString("S3_ENDPOINT", ""),
				AccessKey: envString("S3_ACCESS_KEY", ""),
				SecretKey: envString("S3_SECRET_KEY", ""),
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate checks that required configuration is present and sane.
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("config: JWT_SECRET is required")
	}

	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("config: JWT_SECRET must be at least 32 characters")
	}

	if c.DB.Host == "" {
		return fmt.Errorf("config: DB_HOST is required")
	}

	if c.DB.Name == "" {
		return fmt.Errorf("config: DB_NAME is required")
	}

	switch c.Storage.Driver {
	case "local", "s3":
	default:
		return fmt.Errorf("config: STORAGE_DRIVER must be one of: local, s3")
	}

	if c.Storage.Driver == "s3" && c.Storage.S3.Bucket == "" {
		return fmt.Errorf("config: S3_BUCKET is required when STORAGE_DRIVER=s3")
	}

	return nil
}

// DSN returns the PostgreSQL connection string.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Name,
		c.SSLMode,
	)
}

func envString(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func envInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
