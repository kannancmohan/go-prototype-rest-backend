package main

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/elasticsearch"
)

type Server struct {
	kafka       *kafka.Consumer
	searchStore store.PostSearchIndexStore
	doneC       chan struct{}
	closeC      chan struct{}
}

func main() {

	envName := app_common.GetEnvNameFromCommandLine()
	env := initEnvVar(envName)
	initLogger(env) // error ignored on purpose
	es, err := initElasticSearch(env)
	if err != nil {
		log.Fatalf("Error init ElasticSearch: %s", err)
	}
	kafka, err := initKafkaConsumer(env)
	if err != nil {
		log.Fatalf("Error init KafkaConsumer: %s", err)
	}

	searchStore := elasticsearch.NewPostSearchIndexStore(es, env.ElasticIndexName)

	s := &Server{
		kafka:       kafka,
		searchStore: searchStore,
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

				ok = false

				switch evt.Type {
				case "posts.event.updated", "posts.event.created":
					if err := s.searchStore.Index(context.Background(), evt.Value); err == nil {
						ok = true
					}
				case "posts.event.deleted":
					if err := s.searchStore.Delete(context.Background(), strconv.FormatInt(evt.Value.ID, 10)); err == nil {
						ok = true
					}
				}

				if ok {
					slog.Info("consumed kafka post msg", "type", evt.Type, "postId", evt.Value.ID)
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

func initKafkaConsumer(env *EnvVar) (*kafka.Consumer, error) {
	kafkaCons := app_common.KafkaConsumerConfig{
		Addr:             env.KafkaHost,
		GroupID:          "elasticsearch-indexer",
		AutoOffsetRest:   "earliest",
		EnableAutoCommit: false,
		Topics:           []string{env.KafkaConsumerTopic},
	}
	p, err := kafkaCons.NewKafkaConsumer()
	if err != nil {
		return nil, err
	}
	return p, nil
}
