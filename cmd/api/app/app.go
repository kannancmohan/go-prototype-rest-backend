package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
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

func ListenAndServe(ctx context.Context, envName string, stopChannels ...app_common.StopChan) (err error) {
	defer func() { // Recover from panics and log the stack trace
		if r := recover(); r != nil {
			stackTrace := string(debug.Stack()) // Capture the stack trace
			slog.Error("Recovered from panic", "panic", r, "stack_trace", stackTrace)
			err = fmt.Errorf("panic occurred: %v", r) // Return the panic as an error
		}
	}()

	infra, err := initInfraResources(ctx, envName)
	if err != nil {
		return fmt.Errorf("error initializing infra resources: %w", err)
	}
	defer infra.Close() // Ensure resources are closed when main exits

	store, err := initStoreResources(ctx, infra)
	if err != nil {
		return fmt.Errorf("error initializing store resources: %w", err)
	}

	pService := service.NewPostService(store.postStore, store.msgBrokerStore, store.searchStore)
	uService := service.NewUserService(store.userStore)

	handler := handler.NewHandler(uService, pService)

	router := api.NewRouter(handler, infra.env.AppCorsAllowedOrigin)
	routes := router.RegisterHandlers()

	appServer := newAppServer(infra.env.AppAddr, routes)
	appServer.listenForStopChannels(ctx, stopChannels...)
	if err := appServer.start(); err != nil {
		return err
	}

	return nil
}

type appServer struct {
	name        string
	appStopChan app_common.AppStopChan // used for signalling app shutdown
	httpServer  *http.Server
}

func newAppServer(appAddr string, routes http.Handler) *appServer {
	httpServer := &http.Server{
		Addr:         appAddr,
		Handler:      routes,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}
	return &appServer{name: "api", appStopChan: make(chan struct{}), httpServer: httpServer}
}

func (a *appServer) listenForStopChannels(ctx context.Context, stopChannels ...app_common.StopChan) {
	go func() {
		<-app_common.WaitForStopChan(ctx, stopChannels)
		slog.Debug("external stop signal received in ListenForStopSignals")
		//TODO check usage of a.appStopChan <- struct{}{} instead of close(a.appStopChan)
		close(a.appStopChan) // send app stop signal
	}()
}

func (a *appServer) start() error {
	serverStartErrChan := make(chan error, 1) // Capture errors from HTTP server or shutdown
	go func() {
		slog.Info(fmt.Sprintf("app listening on host: %s", a.httpServer.Addr))
		// After Shutdown or Close, the returned error is ErrServerClosed
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverStartErrChan <- err
		}
	}()

	select {
	case <-a.appStopChan: // on receiving app stop signal, gracefully shut down the server
		slog.Info("stop signal received, shutting down server...")
		return a.stop(context.Background())
	case err := <-serverStartErrChan: // If the server start fails for some reason
		slog.Info("server start error, stopping server(if started)..")
		a.stop(context.Background())
		return fmt.Errorf("server start error: %w", err)
	}
}

func (a *appServer) stop(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	a.httpServer.SetKeepAlivesEnabled(false)
	if err := a.httpServer.Shutdown(timeoutCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		return fmt.Errorf("server shutdown error: %w", err)
	}
	slog.Info("server gracefully stopped")
	return nil
}

type infraResource struct {
	env           *EnvVar
	db            *sql.DB
	redis         *redis.Client
	kafkaProd     *kafka.Producer
	elasticsearch *esv8.Client
}

func newInfraResource(env *EnvVar, db *sql.DB, redis *redis.Client, kafkaProd *kafka.Producer, elasticsearch *esv8.Client) *infraResource {
	return &infraResource{env: env, db: db, redis: redis, kafkaProd: kafkaProd, elasticsearch: elasticsearch}
}

func (i *infraResource) Close() {
	if i.db != nil {
		slog.Debug("Closing db connection")
		i.db.Close()
	}
	if i.redis != nil {
		slog.Debug("Redis client connection closed")
		i.redis.Close()
	}
	slog.Info("finished infra connection cleanup")
}

type storeResource struct {
	postStore      store.PostDBStore
	userStore      store.UserDBStore
	msgBrokerStore store.PostMessageBrokerStore
	searchStore    store.PostSearchStore
}

func newStoreResource(postStore store.PostDBStore, userStore store.UserDBStore, msgBrokerStore store.PostMessageBrokerStore, searchStore store.PostSearchStore) storeResource {
	return storeResource{postStore: postStore, userStore: userStore, msgBrokerStore: msgBrokerStore, searchStore: searchStore}
}

func initStoreResources(ctx context.Context, infra *infraResource) (storeResource, error) {
	pStore := postgres.NewPostDBStore(infra.db, infra.env.AppDBQueryTimeoutDuration)
	uStore := postgres.NewUserDBStore(infra.db, infra.env.AppDBQueryTimeoutDuration)
	//rStore := store.NewRoleStore(db)

	messageBrokerStore := infrastructure_kafka.NewPostMessageBrokerStore(infra.kafkaProd, infra.env.AppKafkaProdTopic)

	cachedPStore := redis_cache.NewPostStore(infra.redis, pStore, infra.env.AppRedisCacheExpirationDuration)
	cachedUStore := redis_cache.NewUserStore(infra.redis, uStore, infra.env.AppRedisCacheExpirationDuration)

	searchStore, err := elasticsearch.NewPostSearchIndexStore(ctx, infra.elasticsearch, infra.env.ElasticIndexName, infra.env.AutoCreateMissingElasticIndex)
	if err != nil {
		return storeResource{}, fmt.Errorf("error init PostSearchIndexStore: %w", err)
	}
	return newStoreResource(cachedPStore, cachedUStore, messageBrokerStore, searchStore), nil
}

func initInfraResources(ctx context.Context, envName string) (*infraResource, error) {

	//TODO run the following processes in goroutine
	//get secrets
	secretStore := envvarsecret.NewSecretFetchStore(envName)
	env := initEnvVar(secretStore)

	// set logger
	err := initLogger(env)
	if err != nil {
		return nil, fmt.Errorf("error init secret: %w", err)
	}

	//database
	dbCfg := app_common.DBConfig{
		Addr:         fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", env.DBUser, env.DBPass, env.DBHost, env.DBPort, env.AppDBName, env.DBSslMode),
		MaxOpenConns: env.AppDBMaxOpenConns,
		MaxIdleConns: env.AppDBMaxIdleConns,
		MaxIdleTime:  env.AppDBMaxIdleTime,
	}
	db, err := dbCfg.NewConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("error init db: %w", err)
	}

	//redis
	redisDB, _ := strconv.Atoi(env.RedisDB)
	redisCfg := app_common.RedisConfig{
		Addr: env.RedisHost,
		DB:   redisDB,
	}
	redis, err := redisCfg.NewConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("error init redis: %w", err)
	}

	//kafka
	kafkaProdCfg := app_common.KafkaProducerConfig{
		Addr: env.KafkaHost,
	}
	kafkaProd, err := kafkaProdCfg.NewKafkaProducer()
	if err != nil {
		return nil, fmt.Errorf("error init kafka producer: %w", err)
	}

	//elastic
	esConfig := app_common.ElasticSearchConfig{
		Addr: env.ElasticHost,
	}
	es, err := esConfig.NewConnection()
	if err != nil {
		return nil, fmt.Errorf("error init ElasticSearch: %w", err)
	}

	return newInfraResource(env, db, redis, kafkaProd, es), nil
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
