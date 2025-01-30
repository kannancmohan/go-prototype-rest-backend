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
	"strconv"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	esv8 "github.com/elastic/go-elasticsearch/v8"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/service"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
	redis_cache "github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/cache/redis"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/db/postgres"
	infrastructure_kafka "github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/messagebroker/kafka"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/search/elasticsearch"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/secret/envvarsecret"
	"github.com/redis/go-redis/v9"
)

func main() {
	infra, err := initInfraResources()
	if err != nil {
		log.Fatalf("Error init infra resources: %s", err.Error())
	}

	store, err := initStoreResources(infra)
	if err != nil {
		log.Fatalf("Error init store resources: %s", err.Error())
	}

	s, _ := newServer(infra.env, store.postStore, store.userStore, store.msgBrokerStore, store.searchStore)
	errC := make(chan error, 1)                    //channel to capture error while start/kill application
	handleShutdown(s, infra.db, infra.redis, errC) //gracefully shutting down applications in response to system signals
	startServer(s, errC)
	if err := <-errC; err != nil {
		log.Fatalf("Error while running: %s", err)
	}
}

func newServer(env *EnvVar, pStore store.PostDBStore, uStore store.UserDBStore, messageBrokerStore store.PostMessageBrokerStore, searchStore store.PostSearchStore) (*http.Server, error) {
	pService := service.NewPostService(pStore, messageBrokerStore, searchStore)
	uService := service.NewUserService(uStore)

	handler := handler.NewHandler(uService, pService)

	router := api.NewRouter(handler, env.ApiCorsAllowedOrigin)
	routes := router.RegisterHandlers()
	return &http.Server{
		Addr:         env.ApiAddr,
		Handler:      routes,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}, nil
}

func startServer(s *http.Server, errC chan<- error) {
	go func() {
		slog.Info(fmt.Sprintf("Listening on host: %s", s.Addr))
		// After Shutdown or Close, the returned error is ErrServerClosed
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errC <- err
		}
	}()
}

func handleShutdown(s *http.Server, db *sql.DB, redis *redis.Client, errC chan<- error) {
	// create notification context that terminates if one of the mentioned signal(eg os.Interrup) is triggered
	ntyCtx, ntyStop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-ntyCtx.Done() // Block until any interrupt signal is received
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

func initStoreResources(infra *infraResource) (storeResource, error) {
	pStore := postgres.NewPostDBStore(infra.db, infra.env.ApiDBQueryTimeoutDuration)
	uStore := postgres.NewUserDBStore(infra.db, infra.env.ApiDBQueryTimeoutDuration)
	//rStore := store.NewRoleStore(db)

	messageBrokerStore := infrastructure_kafka.NewPostMessageBrokerStore(infra.kafkaProd, infra.env.KafkaProdTopic)

	cachedPStore := redis_cache.NewPostStore(infra.redis, pStore, infra.env.ApiRedisCacheExpirationDuration)
	cachedUStore := redis_cache.NewUserStore(infra.redis, uStore, infra.env.ApiRedisCacheExpirationDuration)

	searchStore, err := elasticsearch.NewPostSearchIndexStore(infra.elasticsearch, infra.env.ElasticIndexName)
	if err != nil {
		return storeResource{}, fmt.Errorf("Error init PostSearchIndexStore: %w", err)
	}
	return NewStoreResource(cachedPStore, cachedUStore, messageBrokerStore, searchStore), nil
}

func initInfraResources() (*infraResource, error) {
	env, err := initSecret(app_common.GetEnvNameFromCommandLine())
	if err != nil {
		return nil, fmt.Errorf("Error init secret: %w", err)
	}

	initLogger(env) // error ignored on purpose

	db, err := initDB(env)
	if err != nil {
		return nil, fmt.Errorf("Error init db: %w", err)
	}

	redis, err := initRedis(env)
	if err != nil {
		return nil, fmt.Errorf("Error init redis: %w", err)
	}

	kafkaProd, err := initKafkaProducer(env)
	if err != nil {
		return nil, fmt.Errorf("Error init kafka producer: %w", err)
	}

	es, err := initElasticSearch(env)
	if err != nil {
		return nil, fmt.Errorf("Error init ElasticSearch: %w", err)
	}

	return NewInfraResource(env, db, redis, kafkaProd, es), nil
}

func initSecret(envFileName string) (*EnvVar, error) {
	secretStore := envvarsecret.NewSecretFetchStore(envFileName)
	return initEnvVar(secretStore), nil
}

func initLogger(env *EnvVar) error {
	var level slog.Level
	err := level.UnmarshalText([]byte(env.LogLevel))
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	return nil
}

func initDB(env *EnvVar) (*sql.DB, error) {
	dbCfg := app_common.DBConfig{
		Addr:         fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", env.DBUser, env.DBPass, env.DBHost, env.DBPort, env.ApiDBName, env.DBSslMode),
		MaxOpenConns: env.ApiDBMaxOpenConns,
		MaxIdleConns: env.ApiDBMaxIdleConns,
		MaxIdleTime:  env.ApiDBMaxIdleTime,
	}
	db, err := dbCfg.NewConnection()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func initRedis(env *EnvVar) (*redis.Client, error) {
	host := env.RedisHost
	db := env.RedisDB
	dbi, _ := strconv.Atoi(db)
	rdb := redis.NewClient(&redis.Options{
		Addr: host,
		DB:   dbi,
	})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}
	return rdb, nil
}

func initKafkaProducer(env *EnvVar) (*kafka.Producer, error) {
	kafkaProd := app_common.KafkaProducerConfig{
		Addr: env.KafkaHost,
	}
	p, err := kafkaProd.NewKafkaProducer()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func initElasticSearch(env *EnvVar) (*esv8.Client, error) {
	esConfig := app_common.ElasticSearchConfig{
		Addr: env.ElasticHost,
	}
	es, err := esConfig.NewElasticSearch()
	if err != nil {
		return nil, err
	}
	return es, nil
}

type infraResource struct {
	env           *EnvVar
	db            *sql.DB
	redis         *redis.Client
	kafkaProd     *kafka.Producer
	elasticsearch *esv8.Client
}

func NewInfraResource(env *EnvVar, db *sql.DB, redis *redis.Client, kafkaProd *kafka.Producer, elasticsearch *esv8.Client) *infraResource {
	return &infraResource{env: env, db: db, redis: redis, kafkaProd: kafkaProd, elasticsearch: elasticsearch}
}

type storeResource struct {
	postStore      store.PostDBStore
	userStore      store.UserDBStore
	msgBrokerStore store.PostMessageBrokerStore
	searchStore    store.PostSearchStore
}

func NewStoreResource(postStore store.PostDBStore, userStore store.UserDBStore, msgBrokerStore store.PostMessageBrokerStore, searchStore store.PostSearchStore) storeResource {
	return storeResource{postStore: postStore, userStore: userStore, msgBrokerStore: msgBrokerStore, searchStore: searchStore}
}
