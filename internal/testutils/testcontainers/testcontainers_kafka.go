package testcontainers_testutils

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	kafkatc "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/wait"
)

func CreateKafkaProducer(broker string) (*kafka.Producer, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
	})
	if err != nil {
		return nil, err
	}
	return producer, nil
}

func CreateKafkaConsumer(broker, group string, topics []string) (*kafka.Consumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  broker,
		"group.id":           group,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false,
	})
	if err != nil {
		return nil, err
	}
	if err := consumer.SubscribeTopics(topics, nil); err != nil {
		return nil, err
	}
	return consumer, nil
}

func VerifyKafkaMessage[V any](t *testing.T, consumer *kafka.Consumer, expectedMsgType string, expectedMsg V) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for Kafka message")
		default:
			ev := consumer.Poll(100)
			switch msg := ev.(type) {
			case *kafka.Message:
				var receivedEvent struct {
					Type  string
					Value V
				}
				err := json.Unmarshal(msg.Value, &receivedEvent)
				if err != nil {
					t.Errorf("Failed to unmarshal Kafka event: %v", err)
					return
				}
				if expectedMsgType != receivedEvent.Type {
					t.Errorf("Unexpected message type, expected:%s received:%s", expectedMsgType, receivedEvent.Type)
					return
				}
				if !reflect.DeepEqual(expectedMsg, receivedEvent.Value) {
					t.Errorf("Unexpected message content, expected:%v received:%v", expectedMsg, receivedEvent.Value)
					return
				}
				return
			case kafka.Error:
				t.Fatalf("Error consuming Kafka message: %v", msg)
			}
		}
	}
}

type testKafkaContainer struct {
	clusterID string
}

func NewTestKafkaContainer(clusterID string) *testKafkaContainer {
	return &testKafkaContainer{clusterID: clusterID}
}

func (e *testKafkaContainer) CreateKafkaTestContainer(ctx context.Context) (*kafkatc.KafkaContainer, func(ctx context.Context) error, error) {
	tCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ctr, err := kafkatc.Run(tCtx,
		"confluentinc/confluent-local:7.8.0",
		kafkatc.WithClusterID(e.clusterID),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort(nat.Port("9093/tcp")).SkipInternalCheck().WithStartupTimeout(30*time.Second),
		),
	)

	if err != nil {
		return ctr, func(ctx context.Context) error { return nil }, err
	}

	cleanupFunc := func(ctx context.Context) error {
		err := ctr.Terminate(ctx)
		if err != nil {
			return err
		}
		return nil
	}
	return ctr, cleanupFunc, nil
}
