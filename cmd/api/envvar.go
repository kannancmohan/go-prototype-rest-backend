package main

import (
	"fmt"
	"time"

	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
)

type ApiEnvVar struct {
	ApiAddr                         string
	LogLevel                        string
	DBHost                          string
	DBPort                          string
	DBUser                          string
	DBPass                          string
	DBSslMode                       string
	ApiDBName                       string
	ApiDBQueryTimeoutDuration       time.Duration
	ApiDBMaxOpenConns               int
	ApiDBMaxIdleConns               int
	ApiDBMaxIdleTime                time.Duration
	ApiCorsAllowedOrigin            string
	MemCacheDHost                   string
	RedisHost                       string
	RedisDB                         string
	ApiRedisCacheExpirationDuration time.Duration
	KafkaHost                       string
	KafkaProdTopic                  string
}

// string representation to hide sensitive fields.
func (e ApiEnvVar) String() string {
	return fmt.Sprintf("EnvVar{ApiAddr: %s, LogLevel: %s, DBHost: %s, DBPort: %s, DBUser: [REDACTED], DBPass: [REDACTED], DBSslMode: [REDACTED], ApiCorsAllowedOrigin: %s, MemCacheDHost: %s, RedisHost: %s, RedisDB: %s}", e.ApiAddr, e.LogLevel, e.DBHost, e.DBPort, e.ApiCorsAllowedOrigin, e.MemCacheDHost, e.RedisHost, e.RedisDB)
}

func initApiEnvVar(envName string) *ApiEnvVar {
	env := app_common.NewEnvVarFetcher(envName, nil)
	return &ApiEnvVar{
		ApiAddr:                   fmt.Sprintf(":%s", env.GetEnvString("PORT", "8080")),
		LogLevel:                  env.GetEnvString("LOG_LEVEL", "info"), // supported values DEBUG,INFO,WARN,ERROR
		DBHost:                    env.GetEnvString("DB_HOST", "192.168.0.30"),
		DBPort:                    env.GetEnvString("DB_PORT", "5432"),
		DBUser:                    env.GetEnvString("DB_USER", "admin"),
		DBPass:                    env.GetEnvString("DB_PASS", "adminpassword"),
		DBSslMode:                 env.GetEnvString("DB_SSL_MODE", "disable"),
		ApiDBName:                 env.GetEnvString("API_DB_SCHEMA_NAME", "socialnetwork"),
		ApiDBQueryTimeoutDuration: env.GetEnvDuration("API_DB_QUERY_TIMEOUT", "5s"),
		//ApiDBMaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 30),
		//ApiDBMaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 30),
		ApiDBMaxIdleTime:                env.GetEnvDuration("DB_MAX_IDLE_TIME", "15m"),
		ApiCorsAllowedOrigin:            env.GetEnvString("CORS_ALLOWED_ORIGIN", "http://localhost:8080"),
		MemCacheDHost:                   env.GetEnvString("MEMCACHED_HOST", "192.168.0.30:11211"),
		RedisHost:                       env.GetEnvString("REDIS_HOST", "192.168.0.30:6379"),
		RedisDB:                         env.GetEnvString("REDIS_DB", "socialnetwork"),
		ApiRedisCacheExpirationDuration: env.GetEnvDuration("API_REDIS_CACHE_EXPIRATION", "5m"),
		KafkaHost:                       env.GetEnvString("KAFKA_HOST", "192.168.0.30:9093"),
		KafkaProdTopic:                  env.GetEnvString("API_KAFKA_TOPIC", "posts"),
	}
}
