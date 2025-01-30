package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
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

type Server struct {
	kafka       *kafka.Consumer
	searchStore store.PostSearchIndexStore
	doneC       chan struct{}
	closeC      chan struct{}
}

func main() {

	infra, err := initInfraResources(app_common.GetEnvNameFromCommandLine())
	if err != nil {
		log.Fatalf("Error init infra resources: %s", err.Error())
	}

	store, err := initStoreResources(infra)
	if err != nil {
		log.Fatalf("Error init store resources: %s", err.Error())
	}

	s := &Server{
		kafka:       infra.kafkaCons,
		searchStore: store.searchStore,
		doneC:       make(chan struct{}),
		closeC:      make(chan struct{}),
	}
	errC := make(chan error, 1) //channel to capture error while start/kill application
	handleShutdown(s, errC)     //gracefully shutting down applications in response to system signals
	startServer(s, errC)
	if err := <-errC; err != nil {
		log.Fatalf("Error while running: %s", err)
	}
}

func startServer(s *Server, errC chan<- error) {
	go func() {
		slog.Info("Listening and serving")
		if err := s.ListenAndServe(); err != nil {
			errC <- err
		}
	}()
}

func (s *Server) ListenAndServe() error {
	commit := func(msg *kafka.Message) {
		if _, err := s.kafka.CommitMessage(msg); err != nil {
			slog.Error("kafka 'post' msg commit failed", "error", err)
		}
	}

	handleIndexEvent := func(ctx context.Context, post model.Post) error {
		if err := s.searchStore.Index(ctx, post); err != nil {
			slog.Error("failed to index post", "postID", post.ID, "error", err)
			return err
		}
		return nil
	}

	handleDeleteEvent := func(ctx context.Context, postID int64) error {
		if err := s.searchStore.Delete(ctx, strconv.FormatInt(postID, 10)); err != nil {
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
				msg, ok := s.kafka.Poll(150).(*kafka.Message)
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

func handleShutdown(s *Server, errC chan<- error) {
	// create notification context that terminates if one of the mentioned signal(eg os.Interrup) is triggered
	termSigCtx, termSigCtxCancelFunc := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-termSigCtx.Done() // Block until any interrupt signal is received

		slog.Info("Shutdown signal received")
		timeoutCtx, timeoutCtxCancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
		defer func() {
			s.kafka.Close()
			termSigCtxCancelFunc()
			timeoutCtxCancelFunc()
			close(errC) //close the errC channel
		}()

		// Shutdown the server
		if err := s.Shutdown(timeoutCtx); err != nil {
			errC <- err //log shutdown error if any
		}
		slog.Info("Shutdown completed")

	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	//slog.Info("Shutting down server")
	close(s.closeC)
	for {
		select {
		case <-ctx.Done(): //throw err in case the ctx timeout
			return ctx.Err()
		case <-s.doneC:
			return nil
		}
	}
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

type infraResource struct {
	env           *EnvVar
	kafkaCons     *kafka.Consumer
	elasticsearch *esv8.Client
}

func NewInfraResource(env *EnvVar, kafkaCons *kafka.Consumer, elasticsearch *esv8.Client) *infraResource {
	return &infraResource{env: env, kafkaCons: kafkaCons, elasticsearch: elasticsearch}
}

type storeResource struct {
	searchStore store.PostSearchIndexStore
}

func NewStoreResource(searchStore store.PostSearchIndexStore) storeResource {
	return storeResource{searchStore: searchStore}
}
