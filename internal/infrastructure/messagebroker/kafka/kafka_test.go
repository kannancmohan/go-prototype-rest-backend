//go:build !skip_docker_tests

package kafka_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	infrastructure_kafka "github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/messagebroker/kafka"
	tc_testutils "github.com/kannancmohan/go-prototype-rest-backend/internal/testutils/testcontainers"
)

const (
	kafkaClusterId string = "test_kafka"
	testTopic      string = "test_posts_topic"
)

var (
	consumer *kafka.Consumer
	producer *kafka.Producer
)

func TestMain(m *testing.M) {
	var err error
	pgTest := tc_testutils.NewTestKafkaContainer(kafkaClusterId)
	container, cleanupFunc, err := pgTest.CreateKafkaTestContainer(context.Background())
	if err != nil {
		log.Fatalf("Failed to start kafka TestContainer: %v", err)
	}
	broker, err := pgTest.GetKafkaBrokerAddress(container)
	if err != nil {
		log.Fatalf("Failed to get kafka broker: %v", err)
	}

	producer, err = tc_testutils.CreateKafkaProducer(broker)
	if err != nil {
		log.Fatalf("failed to create Kafka producer: %v", err)
	}

	consumer, err = tc_testutils.CreateKafkaConsumer(broker, "test-group", []string{testTopic})
	if err != nil {
		log.Fatalf("failed to create Kafka consumer: %v", err)
	}

	code := m.Run()

	if producer != nil {
		producer.Close()
	}
	if consumer != nil {
		consumer.Close()
	}

	if cleanupFunc != nil {
		if err := cleanupFunc(context.Background()); err != nil {
			log.Printf("Failed to clean up TestContainer: %v", err)
		}
	}

	os.Exit(code)
}

func TestPostMessageBrokerStore_Created(t *testing.T) {
	testCases := []struct {
		name      string
		eventType string
		event     model.Post
		expErr    error
	}{
		{
			name:      "success",
			eventType: "posts.event.created",
			event: model.Post{
				ID:      1,
				Title:   "test-title",
				Content: "test-content",
				UserID:  1,
			},
		},
	}

	msgStore := infrastructure_kafka.NewPostMessageBrokerStore(producer, testTopic)
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := msgStore.Created(ctx, tc.event)
			if err != nil {
				t.Error("failed to publish Created event", err.Error())
			}
			tc_testutils.VerifyKafkaMessage(t, consumer, tc.eventType, tc.event)
		})
	}
}
