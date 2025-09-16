package config

import (
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// Config holds application configuration
type Config struct {
	Port                         string
	WorkerCount                  int
	RedisURL                     string
	AzureStorageConnectionString string
	MaxFileSize                  int64
	ConversionTimeout            time.Duration
	LogLevel                     string
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		Port:                         getEnv("PORT", "8080"),
		WorkerCount:                  getEnvAsInt("WORKER_COUNT", 10),
		RedisURL:                     getEnv("REDIS_URL", "redis://localhost:6379"),
		AzureStorageConnectionString: getEnv("AZURE_STORAGE_CONNECTION_STRING", ""),
		MaxFileSize:                  getEnvAsInt64("MAX_FILE_SIZE", 50) * 1024 * 1024, // MB to bytes
		ConversionTimeout:            time.Duration(getEnvAsInt("CONVERSION_TIMEOUT", 60)) * time.Second,
		LogLevel:                     getEnv("LOG_LEVEL", "info"),
	}

	log.WithFields(log.Fields{
		"port":        cfg.Port,
		"workers":     cfg.WorkerCount,
		"timeout":     cfg.ConversionTimeout,
		"maxFileSize": cfg.MaxFileSize / 1024 / 1024,
	}).Info("Configuration loaded")

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}