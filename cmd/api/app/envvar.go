package app

import (
	"fmt"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
)

type EnvVar struct {
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
	RedisHost                       string
	RedisDB                         string
	ApiRedisCacheExpirationDuration time.Duration
	KafkaHost                       string
	KafkaProdTopic                  string
	ElasticHost                     string
	ElasticIndexName                string
}

// string representation to hide sensitive fields.
func (e EnvVar) String() string {
	return fmt.Sprintf("EnvVar{ApiAddr: %s, LogLevel: %s, DBHost: %s, DBPort: %s, DBUser: [REDACTED], DBPass: [REDACTED], DBSslMode: [REDACTED], ApiCorsAllowedOrigin: %s, RedisHost: %s, RedisDB: %s}", e.ApiAddr, e.LogLevel, e.DBHost, e.DBPort, e.ApiCorsAllowedOrigin, e.RedisHost, e.RedisDB)
}

func initEnvVar(sec store.SecretFetchStore) *EnvVar {
	return &EnvVar{
		ApiAddr:                   fmt.Sprintf(":%s", sec.GetEnvString("API_PORT", "8080")),
		LogLevel:                  sec.GetEnvString("LOG_LEVEL", "info"), // supported values DEBUG,INFO,WARN,ERROR
		DBHost:                    sec.GetEnvString("DB_HOST", "192.168.0.30"),
		DBPort:                    sec.GetEnvString("DB_PORT", "5432"),
		DBUser:                    sec.GetEnvString("DB_USER", "admin"),
		DBPass:                    sec.GetEnvString("DB_PASS", "adminpassword"),
		DBSslMode:                 sec.GetEnvString("DB_SSL_MODE", "disable"),
		ApiDBName:                 sec.GetEnvString("API_DB_SCHEMA_NAME", "socialnetwork"),
		ApiDBQueryTimeoutDuration: sec.GetEnvDuration("API_DB_QUERY_TIMEOUT", "5s"),
		//ApiDBMaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 30),
		//ApiDBMaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 30),
		ApiDBMaxIdleTime:                sec.GetEnvDuration("DB_MAX_IDLE_TIME", "15m"),
		ApiCorsAllowedOrigin:            sec.GetEnvString("CORS_ALLOWED_ORIGIN", "http://localhost:8080"),
		RedisHost:                       sec.GetEnvString("REDIS_HOST", "192.168.0.30:6379"),
		RedisDB:                         sec.GetEnvString("REDIS_DB", "socialnetwork"),
		ApiRedisCacheExpirationDuration: sec.GetEnvDuration("API_REDIS_CACHE_EXPIRATION", "5m"),
		KafkaHost:                       sec.GetEnvString("KAFKA_HOST", "192.168.0.30:9093"),
		KafkaProdTopic:                  sec.GetEnvString("API_KAFKA_TOPIC", "posts"),
		ElasticHost:                     sec.GetEnvString("ELASTIC_HOST", "http://192.168.0.30:9200"),
		ElasticIndexName:                sec.GetEnvString("ELASTIC_POST_INDEX_NAME", "posts"),
	}
}
