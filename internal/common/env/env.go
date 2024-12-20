package env

import (
	"log/slog"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

// Environment variable common to all apps in this project
type EnvVar struct {
	ApiPort              string
	LogLevel             string
	DBHost               string
	DBPort               string
	DBUser               string
	DBPass               string
	DBSslMode            string
	ApiDBName            string
	ApiDBMaxOpenConns    int
	ApiDBMaxIdleConns    int
	ApiDBMaxIdleTime     string
	ApiCorsAllowedOrigin string
	MemCacheDHost        string
}

var configInstance *EnvVar //configInstance is the cached instance of EnvVar
var once sync.Once

func InitEnvVariables() *EnvVar {
	// Ensure the function runs only once
	once.Do(func() {
		err := godotenv.Load() // Optionally load from a .env file
		if err != nil {
			slog.Info("No .env file found, using system environment variables")
		}

		configInstance = &EnvVar{
			ApiPort:              getString("PORT", "8080"),
			LogLevel:             getString("LOG_LEVEL", "info"), // supported values DEBUG,INFO,WARN,ERROR
			DBHost:               getString("DB_HOST", "192.168.0.30"),
			DBPort:               getString("DB_PORT", "5432"),
			DBUser:               getString("DB_USER", "admin"),
			DBPass:               getString("DB_PASS", "adminpassword"),
			DBSslMode:            getString("DB_SSL_MODE", "disable"),
			ApiDBName:            getString("API_DB_SCHEMA_NAME", "socialnetwork"),
			ApiDBMaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 30),
			ApiDBMaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 30),
			ApiDBMaxIdleTime:     getString("DB_MAX_IDLE_TIME", "15m"),
			ApiCorsAllowedOrigin: getString("CORS_ALLOWED_ORIGIN", "http://localhost:8080"),
			MemCacheDHost:        getString("MEMCACHED_HOST", "192.168.0.30:11211"),
		}
	})
	return configInstance
}

func getString(key, fallback string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return v
}

func getInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	intVal, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return intVal
}

func getBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	boolVal, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return boolVal
}
