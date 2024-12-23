package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/service"
	postgres_memcache "github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/memcache/postgres"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/postgres"
)

func main() {
	env := initApiEnvVar()
	initLogger(env) // error ignored on purpose

	db, err := initDB(env)
	if err != nil {
		log.Fatalf("Error init db: %s", err)
	}

	memCached, err := initMemcached(env)
	if err != nil {
		log.Fatalf("Error init memcached: %s", err)
	}

	conf := &config.ApiConfig{
		Addr:                    fmt.Sprintf(":%s", env.ApiPort),
		CorsAllowedOrigin:       env.ApiCorsAllowedOrigin,
		SqlQueryTimeoutDuration: time.Second * 5,
	}
	s, _ := newServer(conf, db, memCached)
	errC := make(chan error, 1) //channel to capture error while start/kill application
	handleShutdown(s, db, errC) //gracefully shutting down applications in response to system signals
	startServer(s, errC)
	if err := <-errC; err != nil {
		log.Fatalf("Error while running: %s", err)
	}
}

func newServer(cfg *config.ApiConfig, db *sql.DB, memClient *memcache.Client) (*http.Server, error) {

	pStore := postgres.NewPostStore(db, cfg)
	cachedPStore := postgres_memcache.NewPostStore(memClient, pStore, cfg)
	//rStore := store.NewRoleStore(db)
	uStore := postgres.NewUserStore(db, cfg)
	cachedUStore := postgres_memcache.NewUserStore(memClient, uStore, cfg)

	pService := service.NewPostService(cachedPStore)
	uService := service.NewUserService(cachedUStore)

	handler := handler.NewHandler(uService, pService)

	router := api.NewRouter(handler, cfg)
	routes := router.RegisterHandlers()
	return &http.Server{
		Addr:         cfg.Addr,
		Handler:      routes,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}, nil
}

func startServer(s *http.Server, errC chan error) {
	go func() {
		slog.Info(fmt.Sprintf("Listening on host: %s", s.Addr))
		// After Shutdown or Close, the returned error is ErrServerClosed
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errC <- err
		}
	}()
}

func handleShutdown(s *http.Server, db *sql.DB, errC chan error) {
	// create notification context that terminates if one of the mentioned signal(eg os.Interrup) is triggered
	ntyCtx, ntyStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-ntyCtx.Done() // Block until signal is received
		slog.Info("Shutdown signal received")

		ctxTimeout, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer func() {
			db.Close()
			ntyStop()
			ctxCancel()
			close(errC) //close the errC channel
		}()

		// Shutdown the server
		s.SetKeepAlivesEnabled(false)
		if err := s.Shutdown(ctxTimeout); err != nil {
			errC <- err //log shutdown error if any
		}

		slog.Info("Shutdown completed")
	}()
}

func initLogger(env *ApiEnvVar) error {
	var level slog.Level
	err := level.UnmarshalText([]byte(env.LogLevel))
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	return nil
}
