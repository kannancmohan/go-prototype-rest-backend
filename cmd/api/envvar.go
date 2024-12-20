package main

import common_env "github.com/kannancmohan/go-prototype-rest-backend/internal/common/env"

const (
	DBHost  = "DB_HOST"
	DBPort  = "DB_PORT"
	DBUser  = "DB_USER"
	DBPass  = "DB_PASS"
	AppPort = "APP_PORT"
)

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

func initEnvVar() *EnvVar {
	env := common_env.NewEnvVarFetcher("", nil)
	return &EnvVar{
		ApiPort:   env.GetEnvOrFallback("PORT", "8080"),
		LogLevel:  env.GetEnvOrFallback("LOG_LEVEL", "info"), // supported values DEBUG,INFO,WARN,ERROR
		DBHost:    env.GetEnvOrFallback("DB_HOST", "192.168.0.30"),
		DBPort:    env.GetEnvOrFallback("DB_PORT", "5432"),
		DBUser:    env.GetEnvOrFallback("DB_USER", "admin"),
		DBPass:    env.GetEnvOrFallback("DB_PASS", "adminpassword"),
		DBSslMode: env.GetEnvOrFallback("DB_SSL_MODE", "disable"),
		ApiDBName: env.GetEnvOrFallback("API_DB_SCHEMA_NAME", "socialnetwork"),
		//ApiDBMaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 30),
		//ApiDBMaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 30),
		ApiDBMaxIdleTime:     env.GetEnvOrFallback("DB_MAX_IDLE_TIME", "15m"),
		ApiCorsAllowedOrigin: env.GetEnvOrFallback("CORS_ALLOWED_ORIGIN", "http://localhost:8080"),
		MemCacheDHost:        env.GetEnvOrFallback("MEMCACHED_HOST", "192.168.0.30:11211"),
	}
}
