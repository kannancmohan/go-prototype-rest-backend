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

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/service"
	infrastructure_kafka "github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/kafka"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/postgres"
	redis_postgres "github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/redis/postgres"
	"github.com/redis/go-redis/v9"
)

func main() {
	env := initApiEnvVar()
	initLogger(env) // error ignored on purpose

	db, err := initDB(env)
	if err != nil {
		log.Fatalf("Error init db: %s", err)
	}

	redis, err := initRedis(env)
	if err != nil {
		log.Fatalf("Error init redis: %s", err)
	}

	kafkaProd, err := initKafkaProducer(env)
	if err != nil {
		log.Fatalf("Error init kafka producer: %s", err)
	}
	conf := &config.ApiConfig{
		Addr:                    fmt.Sprintf(":%s", env.ApiPort),
		CorsAllowedOrigin:       env.ApiCorsAllowedOrigin,
		SqlQueryTimeoutDuration: time.Second * 5,
		RedisCacheTTL:           time.Minute * 5,
		KafkaProdTopic:          env.KafkaProdTopic,
	}
	s, _ := newServer(conf, db, redis, kafkaProd)
	errC := make(chan error, 1)        //channel to capture error while start/kill application
	handleShutdown(s, db, redis, errC) //gracefully shutting down applications in response to system signals
	startServer(s, errC)
	if err := <-errC; err != nil {
		log.Fatalf("Error while running: %s", err)
	}
}

func newServer(cfg *config.ApiConfig, db *sql.DB, redis *redis.Client, kafkaProd *kafka.Producer) (*http.Server, error) {

	pStore := postgres.NewPostStore(db, cfg.SqlQueryTimeoutDuration)
	uStore := postgres.NewUserStore(db, cfg.SqlQueryTimeoutDuration)
	//rStore := store.NewRoleStore(db)

	cachedPStore := redis_postgres.NewPostStore(redis, pStore, cfg.RedisCacheTTL)
	cachedUStore := redis_postgres.NewUserStore(redis, uStore, cfg.RedisCacheTTL)

	pService := service.NewPostService(cachedPStore, infrastructure_kafka.NewPostMessageBrokerStore(kafkaProd, cfg.KafkaProdTopic))
	uService := service.NewUserService(cachedUStore)

	handler := handler.NewHandler(uService, pService)

	router := api.NewRouter(handler, cfg.CorsAllowedOrigin)
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

func handleShutdown(s *http.Server, db *sql.DB, redis *redis.Client, errC chan error) {
	// create notification context that terminates if one of the mentioned signal(eg os.Interrup) is triggered
	ntyCtx, ntyStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-ntyCtx.Done() // Block until signal is received
		slog.Info("Shutdown signal received")

		ctxTimeout, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeResources(db, redis)
		defer func() {
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

func closeResources(db *sql.DB, redis *redis.Client) {
	if db != nil {
		db.Close()
		slog.Info("Database connection closed")
	}
	if redis != nil {
		redis.Close()
		slog.Info("Redis client connection closed")
	}
}
