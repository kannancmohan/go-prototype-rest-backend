package api

type Api struct {
	Config Config
	// store         store.Storage
	// cacheStorage  cache.Storage
	// logger        *zap.SugaredLogger
	// mailer        mailer.Client
	// authenticator auth.Authenticator
	// rateLimiter   ratelimiter.Limiter
}

type Config struct {
	Addr              string
	CorsAllowedOrigin string
	// db   db.DBConfig
	// env         string
	// apiURL      string
	// mail        mailConfig
	// frontendURL string
	// auth        authConfig
	// redisCfg    redisConfig
	// rateLimiter ratelimiter.Config
}
