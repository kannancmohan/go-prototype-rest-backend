package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	esv8 "github.com/elastic/go-elasticsearch/v8"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/search/elasticsearch"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/secret/envvarsecret"
)

func StartApp(envName string) error {
	infra, err := initInfraResources(envName)
	if err != nil {
		return fmt.Errorf("error initializing infra resources: %w", err)
	}
	defer infra.Close() // Ensure resources are closed when main exits

	store, err := initStoreResources(infra)
	if err != nil {
		return fmt.Errorf("error initializing store resources: %w", err)
	}

	app := NewAppServer(infra, store)
	errC := make(chan error, 1) //channel to capture error while start/kill application
	app.handleShutdown(errC)
	app.start(errC)

	if err := <-errC; err != nil {
		return fmt.Errorf("error while running the application: %w", err)
	}
	return nil
}

func initStoreResources(infra *infraResource) (storeResource, error) {
	searchStore, err := elasticsearch.NewPostSearchIndexStore(infra.elasticsearch, infra.env.ElasticIndexName)
	if err != nil {
		return storeResource{}, fmt.Errorf("Error init PostSearchIndexStore: %w", err)
	}
	return NewStoreResource(searchStore), nil
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

	//kafka
	kafkaConsCfg := app_common.KafkaConsumerConfig{
		Addr:             env.KafkaHost,
		GroupID:          "elasticsearch-indexer",
		AutoOffsetRest:   "earliest",
		EnableAutoCommit: false,
		Topics:           []string{env.KafkaConsumerTopic},
	}
	kafkaCons, err := kafkaConsCfg.NewKafkaConsumer()
	if err != nil {
		return nil, fmt.Errorf("Error init kafka consumer: %w", err)
	}

	//elastic
	esConfig := app_common.ElasticSearchConfig{
		Addr: env.ElasticHost,
	}
	es, err := esConfig.NewConnection()
	if err != nil {
		return nil, fmt.Errorf("Error init ElasticSearch: %w", err)
	}

	return NewInfraResource(env, kafkaCons, es), nil
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
	infra  *infraResource
	store  storeResource
	doneC  chan struct{} // to signal that kafka consumer is done
	closeC chan struct{} // to signal occurrence of interrupt signal to kafka consumer(so as it can stop)
}

func NewAppServer(infra *infraResource, store storeResource) *appServer {
	return &appServer{infra: infra, store: store, doneC: make(chan struct{}), closeC: make(chan struct{})}
}

func (s *appServer) start(errC chan<- error) {
	go func() {
		slog.Info("Listening and serving")
		if err := s.listenAndServe(); err != nil {
			errC <- err
		}
	}()
}

func (s *appServer) listenAndServe() error {
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

	go func() {
		run := true

		for run {
			select {
			case <-s.closeC: // on receiving closeC signal, stop the kafka consumer
				run = false
				break
			default:
				msg, ok := s.infra.kafkaCons.Poll(150).(*kafka.Message)
				if !ok {
					continue
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

		slog.Info("No more messages to consume. Exiting.")
		s.doneC <- struct{}{}
	}()

	return nil
}

func (s *appServer) handleShutdown(errC chan<- error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigChan
		slog.Info("Shutdown signal received", "signal", sig.String())

		timeoutCtx, timeoutCtxCancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer timeoutCtxCancelFunc()

		close(s.closeC) // Notify kafka consumer code that we're shutting down because of interrupt signal

		// Wait for either the timeout or the Kafka consumer to finish
		select {
		case <-timeoutCtx.Done(): // Throw error if the context times out
			errC <- timeoutCtx.Err()
		case <-s.doneC:
		}

		slog.Info("Shutdown completed")
		close(errC) // Close the error channel
	}()
}

type infraResource struct {
	env           *EnvVar
	kafkaCons     *kafka.Consumer
	elasticsearch *esv8.Client
}

func NewInfraResource(env *EnvVar, kafkaCons *kafka.Consumer, elasticsearch *esv8.Client) *infraResource {
	return &infraResource{env: env, kafkaCons: kafkaCons, elasticsearch: elasticsearch}
}

func (i *infraResource) Close() {
	slog.Info("Closing Kafka consumer")
	i.kafkaCons.Close()
}

type storeResource struct {
	searchStore store.PostSearchIndexStore
}

func NewStoreResource(searchStore store.PostSearchIndexStore) storeResource {
	return storeResource{searchStore: searchStore}
}
