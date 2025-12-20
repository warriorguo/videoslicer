package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Worker   WorkerConfig   `json:"worker"`
	Storage  StorageConfig  `json:"storage"`
}

type ServerConfig struct {
	Port        string `json:"port"`
	MaxFileSize int64  `json:"max_file_size"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

type WorkerConfig struct {
	MaxConcurrent int           `json:"max_concurrent"`
	PollInterval  time.Duration `json:"poll_interval"`
	LeaseDuration time.Duration `json:"lease_duration"`
	Timeout       time.Duration `json:"timeout"`
}

type StorageConfig struct {
	TaskDir string `json:"task_dir"`
}

func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:        getEnv("SERVER_PORT", "8080"),
			MaxFileSize: getEnvInt64("MAX_FILE_SIZE", 1024*1024*1024), // 1GB default
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "192.168.0.151"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "liuli"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "postgres"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Worker: WorkerConfig{
			MaxConcurrent: getEnvInt("WORKER_MAX_CONCURRENT", 2),
			PollInterval:  getEnvDuration("WORKER_POLL_INTERVAL", 5*time.Second),
			LeaseDuration: getEnvDuration("WORKER_LEASE_DURATION", 5*time.Minute),
			Timeout:       getEnvDuration("WORKER_TIMEOUT", 30*time.Minute),
		},
		Storage: StorageConfig{
			TaskDir: getEnv("STORAGE_TASK_DIR", "./tasks"),
		},
	}

	// Validate required environment variables
	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) validate() error {
	// Only validate in production environment (when specific env vars are explicitly empty)
	if os.Getenv("DB_HOST") == "" && os.Getenv("REQUIRE_CONFIG_VALIDATION") == "true" {
		return fmt.Errorf("DB_HOST is required")
	}
	if os.Getenv("DB_USER") == "" && os.Getenv("REQUIRE_CONFIG_VALIDATION") == "true" {
		return fmt.Errorf("DB_USER is required")
	}
	if os.Getenv("DB_PASSWORD") == "" && os.Getenv("REQUIRE_CONFIG_VALIDATION") == "true" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if os.Getenv("DB_NAME") == "" && os.Getenv("REQUIRE_CONFIG_VALIDATION") == "true" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.Storage.TaskDir == "" {
		return fmt.Errorf("STORAGE_TASK_DIR is required")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvRequired(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
