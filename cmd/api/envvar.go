package main

import (
	"fmt"

	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
)

type ApiEnvVar struct {
	ApiAddr              string
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
	RedisHost            string
	RedisDB              string
	KafkaHost            string
	KafkaProdTopic       string
}

// string representation to hide sensitive fields.
func (e ApiEnvVar) String() string {
	return fmt.Sprintf("EnvVar{ApiAddr: %s, LogLevel: %s, DBHost: %s, DBPort: %s, DBUser: [REDACTED], DBPass: [REDACTED], DBSslMode: [REDACTED], ApiCorsAllowedOrigin: %s, MemCacheDHost: %s, RedisHost: %s, RedisDB: %s}", e.ApiAddr, e.LogLevel, e.DBHost, e.DBPort, e.ApiCorsAllowedOrigin, e.MemCacheDHost, e.RedisHost, e.RedisDB)
}

func initApiEnvVar() *ApiEnvVar {
	env := app_common.NewEnvVarFetcher("", nil)
	return &ApiEnvVar{
		ApiAddr:   fmt.Sprintf(":%s", env.GetEnvOrFallback("PORT", "8080")),
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
		RedisHost:            env.GetEnvOrFallback("REDIS_HOST", "192.168.0.30:6379"),
		RedisDB:              env.GetEnvOrFallback("REDIS_DB", "socialnetwork"),
		KafkaHost:            env.GetEnvOrFallback("KAFKA_HOST", "192.168.0.30:9093"),
		KafkaProdTopic:       env.GetEnvOrFallback("API_KAFKA_TOPIC", "posts"),
	}
}
