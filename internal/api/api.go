package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/service"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/env"
)

type Api struct {
	config config
	// store         store.Storage
	// cacheStorage  cache.Storage
	// logger        *zap.SugaredLogger
	// mailer        mailer.Client
	// authenticator auth.Authenticator
	// rateLimiter   ratelimiter.Limiter
}

type config struct {
	addr              string
	corsAllowedOrigin string
	// db   db.DBConfig
	// env         string
	// apiURL      string
	// mail        mailConfig
	// frontendURL string
	// auth        authConfig
	// redisCfg    redisConfig
	// rateLimiter ratelimiter.Config
}

func NewAPI() *Api {
	return &Api{
		config: config{
			addr:              env.GetString("ADDR", ":8080"),
			corsAllowedOrigin: env.GetString("CORS_ALLOWED_ORIGIN", "http://localhost:8080"),
		},
	}
}

func (api *Api) Run(db *sql.DB) error {
	store := store.NewStorage(db)
	service := service.NewService(store)
	handler := handler.NewHandler(service)

	router := NewRouter(handler, api.config)
	routes := router.registerHandlers()
	srv := &http.Server{
		Addr:         api.config.addr,
		Handler:      routes,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
