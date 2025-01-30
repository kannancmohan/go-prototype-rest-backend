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
	infra, err := initInfraResources(app_common.GetEnvNameFromCommandLine())
	if err != nil {
		log.Fatalf("Error init infra resources: %s", err.Error())
	}
	defer infra.Close() // Ensure resources are closed when main exits

	store, err := initStoreResources(infra)
	if err != nil {
		log.Fatalf("Error init store resources: %s", err.Error())
	}

	app := NewAppServer(infra, store)
	errC := make(chan error, 1) //channel to capture error while start/kill application
	app.handleShutdown(errC)
	app.start(errC)

	if err := <-errC; err != nil {
		log.Fatalf("Error while running: %s", err)
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

func initInfraResources(envName string) (*infraResource, error) {

	//get secrets
	secretStore := envvarsecret.NewSecretFetchStore(envName)
	env := initEnvVar(secretStore)

	// set logger
	err := initLogger(env)
	if err != nil {
		return nil, fmt.Errorf("Error init secret: %w", err)
	}

	//database
	dbCfg := app_common.DBConfig{
		Addr:         fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", env.DBUser, env.DBPass, env.DBHost, env.DBPort, env.ApiDBName, env.DBSslMode),
		MaxOpenConns: env.ApiDBMaxOpenConns,
		MaxIdleConns: env.ApiDBMaxIdleConns,
		MaxIdleTime:  env.ApiDBMaxIdleTime,
	}
	db, err := dbCfg.NewConnection()
	if err != nil {
		return nil, fmt.Errorf("Error init db: %w", err)
	}

	//redis
	redisDB, _ := strconv.Atoi(env.RedisDB)
	redisCfg := app_common.RedisConfig{
		Addr: env.RedisHost,
		DB:   redisDB,
	}
	redis, err := redisCfg.NewConnection()
	if err != nil {
		return nil, fmt.Errorf("Error init redis: %w", err)
	}

	//kafka
	kafkaProdCfg := app_common.KafkaProducerConfig{
		Addr: env.KafkaHost,
	}
	kafkaProd, err := kafkaProdCfg.NewKafkaProducer()
	if err != nil {
		return nil, fmt.Errorf("Error init kafka producer: %w", err)
	}

	//elastic
	esConfig := app_common.ElasticSearchConfig{
		Addr: env.ElasticHost,
	}
	es, err := esConfig.NewConnection()
	if err != nil {
		return nil, fmt.Errorf("Error init ElasticSearch: %w", err)
	}

	return NewInfraResource(env, db, redis, kafkaProd, es), nil
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

type appServer struct {
	infra      *infraResource
	store      storeResource
	httpServer *http.Server
}

func NewAppServer(infra *infraResource, store storeResource) *appServer {
	pService := service.NewPostService(store.postStore, store.msgBrokerStore, store.searchStore)
	uService := service.NewUserService(store.userStore)

	handler := handler.NewHandler(uService, pService)

	router := api.NewRouter(handler, infra.env.ApiCorsAllowedOrigin)
	routes := router.RegisterHandlers()
	httpServer := &http.Server{
		Addr:         infra.env.ApiAddr,
		Handler:      routes,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}
	return &appServer{infra: infra, store: store, httpServer: httpServer}
}

func (a *appServer) start(errC chan<- error) {
	go func() {
		slog.Info(fmt.Sprintf("Listening on host: %s", a.httpServer.Addr))
		// After Shutdown or Close, the returned error is ErrServerClosed
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errC <- err
		}
	}()
}

func (a *appServer) handleShutdown(errC chan<- error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigChan
		slog.Info("Shutdown signal received", "signal", sig.String())

		ctxTimeout, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer ctxCancel()

		// Shutdown the server
		a.httpServer.SetKeepAlivesEnabled(false)
		if err := a.httpServer.Shutdown(ctxTimeout); err != nil {
			errC <- err //log shutdown error if any
		}

		slog.Info("Shutdown completed")
		close(errC) // Close the error channel
	}()
}

// func closeResources(db *sql.DB, redis *redis.Client) {
// 	if db != nil {
// 		db.Close()
// 		slog.Info("Database connection closed")
// 	}
// 	if redis != nil {
// 		redis.Close()
// 		slog.Info("Redis client connection closed")
// 	}
// }

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

func (i *infraResource) Close() {
	if i.db != nil {
		slog.Info("Closing db connection")
		i.db.Close()
	}
	if i.redis != nil {
		slog.Info("Redis client connection closed")
		i.redis.Close()
	}
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
