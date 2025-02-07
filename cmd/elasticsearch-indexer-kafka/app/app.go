package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	esv8 "github.com/elastic/go-elasticsearch/v8"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/search/elasticsearch"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/secret/envvarsecret"
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

	appServer := newAppServer(infra, store)
	appServer.listenForStopChannels(ctx, stopChannels...)
	if err = appServer.start(); err != nil {
		return err
	}

	return nil
}

type appServer struct {
	infra       *infraResource
	store       storeResource
	appStopChan app_common.AppStopChan // used for signalling app shutdown
}

func newAppServer(infra *infraResource, store storeResource) *appServer {
	return &appServer{infra: infra, store: store, appStopChan: make(chan struct{})}
}

func (a *appServer) listenForStopChannels(ctx context.Context, stopChannels ...app_common.StopChan) {
	go func() {
		<-app_common.WaitForStopChan(ctx, stopChannels)
		slog.Debug("external stop signal received in ListenForStopSignals")
		//TODO check usage of a.appStopChan <- struct{}{} instead of close(a.appStopChan)
		close(a.appStopChan) // send app stop signal
	}()
}

func (s *appServer) start() error {
	commit := func(msg *kafka.Message) {
		if _, err := s.infra.kafkaCons.CommitMessage(msg); err != nil {
			slog.Error("kafka 'post' msg commit failed", "error", err)
		}
	}

	handleIndexEvent := func(ctx context.Context, post model.Post) error {
		if err := s.store.searchStore.Index(ctx, post); err != nil {
			slog.Error("failed to index post", "postID", post.ID, "error", err)
			return err
		}
		return nil
	}

	handleDeleteEvent := func(ctx context.Context, postID int64) error {
		if err := s.store.searchStore.Delete(ctx, strconv.FormatInt(postID, 10)); err != nil {
			slog.Error("failed to delete post", "postID", postID, "error", err)
			return err
		}
		return nil
	}

	for {
		select {
		case <-s.appStopChan: // on receiving stop signal
			slog.Info("stop signal received, shutting down indexer app...")
			return nil // Exits this goroutine
		default:
			msg, ok := s.infra.kafkaCons.Poll(150).(*kafka.Message)
			if !ok {
				continue //if received event is not of desired type, move to the next loop
			}

			var evt struct {
				Type  string
				Value model.Post
			}

			if err := json.NewDecoder(bytes.NewReader(msg.Value)).Decode(&evt); err != nil {
				slog.Warn("failed to marshall kafka msg", "error", err) //TODO add more details
				commit(msg)                                             // here we are committing failed msg
				continue
			}

			ctx := context.Background()
			var err error
			switch evt.Type {
			case "posts.event.updated", "posts.event.created":
				err = handleIndexEvent(ctx, evt.Value)
			case "posts.event.deleted":
				err = handleDeleteEvent(ctx, evt.Value.ID)
			default:
				slog.Warn("unsupported event type", "type", evt.Type)
			}

			if err == nil {
				slog.Info("consumed Kafka post message", "type", evt.Type, "postID", evt.Value.ID)
				commit(msg)
			}
		}
	}
}

type infraResource struct {
	env           *EnvVar
	kafkaCons     *kafka.Consumer
	elasticsearch *esv8.Client
}

func newInfraResource(env *EnvVar, kafkaCons *kafka.Consumer, elasticsearch *esv8.Client) *infraResource {
	return &infraResource{env: env, kafkaCons: kafkaCons, elasticsearch: elasticsearch}
}

func (i *infraResource) Close() {
	slog.Info("closing Kafka consumer")
	i.kafkaCons.Close()
}

type storeResource struct {
	searchStore store.PostSearchIndexStore
}

func newStoreResource(searchStore store.PostSearchIndexStore) storeResource {
	return storeResource{searchStore: searchStore}
}

func initStoreResources(ctx context.Context, infra *infraResource) (storeResource, error) {
	searchStore, err := elasticsearch.NewPostSearchIndexStore(ctx, infra.elasticsearch, infra.env.ElasticIndexName)
	if err != nil {
		return storeResource{}, fmt.Errorf("error init PostSearchIndexStore: %w", err)
	}
	return newStoreResource(searchStore), nil
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

	//kafka
	kafkaConsCfg := app_common.KafkaConsumerConfig{
		Addr:             env.KafkaHost,
		GroupID:          env.KafkaConsumerGroupId,
		AutoOffsetRest:   env.KafkaAutoOffsetRest,
		EnableAutoCommit: false,
		Topics:           []string{env.KafkaConsumerTopic},
	}
	kafkaCons, err := kafkaConsCfg.NewKafkaConsumer()
	if err != nil {
		return nil, fmt.Errorf("error init kafka consumer: %w", err)
	}

	//elastic
	esConfig := app_common.ElasticSearchConfig{
		Addr: env.ElasticHost,
	}
	es, err := esConfig.NewConnection()
	if err != nil {
		return nil, fmt.Errorf("error init ElasticSearch: %w", err)
	}

	return newInfraResource(env, kafkaCons, es), nil
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
