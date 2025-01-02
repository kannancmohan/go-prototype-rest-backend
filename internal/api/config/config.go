package config

import "time"

type ApiConfig struct {
	Addr                    string
	CorsAllowedOrigin       string
	SqlQueryTimeoutDuration time.Duration
	RedisCacheTTL           time.Duration
	KafkaProdTopic          string
	// db   db.DBConfig
	// env         string
	// apiURL      string
	// mail        mailConfig
	// frontendURL string
	// auth        authConfig
	// redisCfg    redisConfig
	// rateLimiter ratelimiter.Config
}
