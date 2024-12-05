package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/env"
)

type Api struct {
	config config
	store  store.Storage
	// store         store.Storage
	// cacheStorage  cache.Storage
	// logger        *zap.SugaredLogger
	// mailer        mailer.Client
	// authenticator auth.Authenticator
	// rateLimiter   ratelimiter.Limiter
}

type config struct {
	addr string
	// db   db.DBConfig
	// env         string
	// apiURL      string
	// mail        mailConfig
	// frontendURL string
	// auth        authConfig
	// redisCfg    redisConfig
	// rateLimiter ratelimiter.Config
}

func NewAPI(db *sql.DB) *Api {
	store := store.NewStorage(db)
	return &Api{
		config: config{
			addr: env.GetString("ADDR", ":8080"),
		},
		store: store,
	}
}

func (api *Api) Run() error {
	routes := registerRouter()
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

func registerRouter() http.Handler {
	return nil
}
