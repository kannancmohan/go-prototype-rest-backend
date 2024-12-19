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

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/service"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/db"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/env"
)

func main() {
	env := env.InitEnvVariables()
	initLogger(env)
	db := initDB(env)
	conf := &config.ApiConfig{
		Addr:                    fmt.Sprintf(":%s", env.ApiPort),
		CorsAllowedOrigin:       env.ApiCorsAllowedOrigin,
		SqlQueryTimeoutDuration: time.Second * 5,
	}
	s, _ := newServer(conf, db)
	errC := make(chan error, 1) //channel to capture error while start/kill application
	handleShutdown(s, db, errC) //gracefully shutting down applications in response to system signals
	startServer(s, errC)
	if err := <-errC; err != nil {
		log.Fatalf("Error while running: %s", err)
	}
}

func newServer(config *config.ApiConfig, db *sql.DB) (*http.Server, error) {
	pStore := store.NewPostStore(db, config)
	//rStore := store.NewRoleStore(db)
	uStore := store.NewUserStore(db, config)

	pService := service.NewPostService(pStore)
	uService := service.NewUserService(uStore)

	handler := handler.NewHandler(uService, pService)

	router := api.NewRouter(handler, config)
	routes := router.RegisterHandlers()
	return &http.Server{
		Addr:         config.Addr,
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

func initLogger(env *env.EnvVar) {
	var level slog.Level
	err := level.UnmarshalText([]byte(env.LogLevel))
	if err != nil {
		log.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
}

func initDB(env *env.EnvVar) *sql.DB {
	dbCfg := db.DBConfig{
		Addr:         fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", env.DBUser, env.DBPass, env.DBHost, env.ApiDBName, env.DBSslMode),
		MaxOpenConns: env.ApiDBMaxOpenConns,
		MaxIdleConns: env.ApiDBMaxIdleConns,
		MaxIdleTime:  env.ApiDBMaxIdleTime,
	}
	db, err := dbCfg.NewConnection()
	if err != nil {
		log.Fatal(err)
	}
	return db
}
