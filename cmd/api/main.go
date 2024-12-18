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
	initLogger()
	dbCfg := db.DBConfig{
		Addr:         env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost/socialnetwork?sslmode=disable"),
		MaxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
		MaxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
		MaxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
	}
	db, err := dbCfg.NewConnection()
	if err != nil {
		log.Fatal(err)
	}

	conf := &config.ApiConfig{
		Addr:                    env.GetString("ADDR", ":8080"),
		CorsAllowedOrigin:       env.GetString("CORS_ALLOWED_ORIGIN", "http://localhost:8080"),
		SqlQueryTimeoutDuration: time.Second * 5,
	}
	s, err := newServer(conf, db)
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

func initLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
}
