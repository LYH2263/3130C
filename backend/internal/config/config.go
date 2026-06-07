package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultJWTSecret = "change-me"

type Config struct {
	AppEnv    string
	LogLevel  string
	Port      string
	JWTSecret string

	DBHost            string
	DBPort            string
	DBUser            string
	DBPass            string
	DBName            string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
}

func Load() Config {
	maxOpen, _ := strconv.Atoi(getOrDefault("DB_MAX_OPEN_CONNS", "25"))
	maxIdle, _ := strconv.Atoi(getOrDefault("DB_MAX_IDLE_CONNS", "10"))
	connMaxLifetime, _ := time.ParseDuration(getOrDefault("DB_CONN_MAX_LIFETIME", "1h"))

	cfg := Config{
		AppEnv:    getOrDefault("APP_ENV", "development"),
		LogLevel:  getOrDefault("LOG_LEVEL", "info"),
		Port:      getOrDefault("PORT", "8130"),
		JWTSecret: getOrDefault("JWT_SECRET", defaultJWTSecret),

		DBHost:            getOrDefault("DB_HOST", "127.0.0.1"),
		DBPort:            getOrDefault("DB_PORT", "3306"),
		DBUser:            getOrDefault("DB_USER", "root"),
		DBPass:            getOrDefault("DB_PASSWORD", "root"),
		DBName:            getOrDefault("DB_NAME", "quizlab"),
		DBMaxOpenConns:    maxOpen,
		DBMaxIdleConns:    maxIdle,
		DBConnMaxLifetime: connMaxLifetime,
	}
	return cfg
}

func (c Config) Validate() error {
	var errs []string

	if strings.TrimSpace(c.JWTSecret) == "" || c.JWTSecret == defaultJWTSecret {
		errs = append(errs, "JWT_SECRET must be set to a non-default, strong value")
	}

	if strings.TrimSpace(c.DBHost) == "" {
		errs = append(errs, "DB_HOST is required")
	}
	if strings.TrimSpace(c.DBPort) == "" {
		errs = append(errs, "DB_PORT is required")
	}
	if strings.TrimSpace(c.DBUser) == "" {
		errs = append(errs, "DB_USER is required")
	}
	if strings.TrimSpace(c.DBName) == "" {
		errs = append(errs, "DB_NAME is required")
	}

	if c.DBMaxOpenConns < 0 {
		errs = append(errs, "DB_MAX_OPEN_CONNS must be non-negative")
	}
	if c.DBMaxIdleConns < 0 {
		errs = append(errs, "DB_MAX_IDLE_CONNS must be non-negative")
	}
	if c.DBMaxIdleConns > c.DBMaxOpenConns && c.DBMaxOpenConns > 0 {
		errs = append(errs, "DB_MAX_IDLE_CONNS must not exceed DB_MAX_OPEN_CONNS")
	}
	if c.DBConnMaxLifetime < 0 {
		errs = append(errs, "DB_CONN_MAX_LIFETIME must be non-negative")
	}

	if len(errs) > 0 {
		return errors.New("configuration validation failed: " + strings.Join(errs, "; "))
	}
	return nil
}

func (c Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser,
		c.DBPass,
		c.DBHost,
		c.DBPort,
		c.DBName,
	)
}

func getOrDefault(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
