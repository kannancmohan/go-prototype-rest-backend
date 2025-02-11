package app

import (
	"fmt"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
)

type EnvVar struct {
	AppAddr                         string
	LogLevel                        string
	DBHost                          string
	DBPort                          string
	DBUser                          string
	DBPass                          string
	DBSslMode                       string
	AppDBName                       string
	AppDBQueryTimeoutDuration       time.Duration
	AppDBMaxOpenConns               int
	AppDBMaxIdleConns               int
	AppDBMaxIdleTime                time.Duration
	AppCorsAllowedOrigin            string
	RedisHost                       string
	RedisDB                         string
	AppRedisCacheExpirationDuration time.Duration
	KafkaHost                       string
	AppKafkaProdTopic               string
	ElasticHost                     string
	ElasticIndexName                string
	AutoCreateMissingElasticIndex   bool
}

// string representation to hide sensitive fields.
func (e EnvVar) String() string {
	return fmt.Sprintf("EnvVar{AppAddr: %s, LogLevel: %s, DBHost: %s, DBPort: %s, DBUser: [REDACTED], DBPass: [REDACTED], DBSslMode: [REDACTED], AppCorsAllowedOrigin: %s, RedisHost: %s, RedisDB: %s}", e.AppAddr, e.LogLevel, e.DBHost, e.DBPort, e.AppCorsAllowedOrigin, e.RedisHost, e.RedisDB)
}

func initEnvVar(sec store.SecretFetchStore) *EnvVar {
	return &EnvVar{
		AppAddr:                   fmt.Sprintf(":%s", sec.GetEnvString("APP_API_PORT", "8080")),
		LogLevel:                  sec.GetEnvString("LOG_LEVEL", "info"), // supported values DEBUG,INFO,WARN,ERROR
		DBHost:                    sec.GetEnvString("DB_HOST", "192.168.0.30"),
		DBPort:                    sec.GetEnvString("DB_PORT", "5432"),
		DBUser:                    sec.GetEnvString("DB_USER", "admin"),
		DBPass:                    sec.GetEnvString("DB_PASS", "adminpassword"),
		DBSslMode:                 sec.GetEnvString("DB_SSL_MODE", "disable"),
		AppDBName:                 sec.GetEnvString("APP_API_DB_SCHEMA_NAME", "socialnetwork"),
		AppDBQueryTimeoutDuration: sec.GetEnvDuration("APP_API_DB_QUERY_TIMEOUT", "5s"),
		//ApiDBMaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 30),
		//ApiDBMaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 30),
		AppDBMaxIdleTime:                sec.GetEnvDuration("APP_API_DB_MAX_IDLE_TIME", "15m"),
		AppCorsAllowedOrigin:            sec.GetEnvString("APP_API_CORS_ALLOWED_ORIGIN", "http://localhost:8080"),
		RedisHost:                       sec.GetEnvString("REDIS_HOST", "192.168.0.30:6379"),
		RedisDB:                         sec.GetEnvString("REDIS_DB", "socialnetwork"),
		AppRedisCacheExpirationDuration: sec.GetEnvDuration("APP_API_REDIS_CACHE_EXPIRATION", "5m"),
		KafkaHost:                       sec.GetEnvString("KAFKA_HOST", "192.168.0.30:9093"),
		AppKafkaProdTopic:               sec.GetEnvString("APP_API_KAFKA_TOPIC", "posts"),
		ElasticHost:                     sec.GetEnvString("ELASTIC_HOST", "http://192.168.0.30:9200"),
		ElasticIndexName:                sec.GetEnvString("ELASTIC_POST_INDEX_NAME", "posts"),
		AutoCreateMissingElasticIndex:   sec.GetEnvBool("ELASTIC_AUTO_CREATE_POST_INDEX", "true"), // if set to true, app will creates the above index if its missing
	}
}
