package testutils

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
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

const kafkaExportedPort string = "9092"

type testKafkaContainer struct {
	clusterID string
}

func NewTestKafkaContainer(clusterID string) *testKafkaContainer {
	return &testKafkaContainer{clusterID: clusterID}
}

func (e *testKafkaContainer) CreateKafkaTestContainer() (testcontainers.Container, func(ctx context.Context) error, error) {
	tcExposedPort := kafkaExportedPort + "/tcp"

	kafkaReq := testcontainers.ContainerRequest{
		Image:        "confluentinc/confluent-local:7.5.0",
		ExposedPorts: []string{tcExposedPort},
		Env: map[string]string{
			"KAFKA_PROCESS_ROLES": "broker,controller",
			// "KAFKA_CONTROLLER_QUORUM_VOTERS":      "1@localhost:9093",
			// "KAFKA_LISTENERS":                     "PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093",
			// "KAFKA_ADVERTISED_LISTENERS":          "PLAINTEXT://192.168.0.30:9092",
			// "KAFKA_CONTROLLER_LISTENER_NAMES":     "CONTROLLER",
			"KAFKA_CLUSTER_ID":                       e.clusterID,
			"KAFKA_AUTO_CREATE_TOPICS_ENABLE":        "true",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR": "1",
			"KAFKA_LOG_RETENTION_HOURS":              "1", // Optional: Shorten log retention for testing
		},
		WaitingFor: wait.ForListeningPort(nat.Port(tcExposedPort)).
			SkipInternalCheck().
			WithStartupTimeout(1 * time.Minute),
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ctr, err := testcontainers.GenericContainer(timeoutCtx, testcontainers.GenericContainerRequest{
		ContainerRequest: kafkaReq,
		Started:          true,
	})

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

func (e *testKafkaContainer) GetKafkaBrokerAddress(container testcontainers.Container) (string, error) {

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := container.MappedPort(timeoutCtx, nat.Port(kafkaExportedPort))
	if err != nil {
		return "", err
	}

	broker, err := container.PortEndpoint(timeoutCtx, nat.Port(kafkaExportedPort+"/tcp"), "")
	if err != nil {
		return "", err
	}
	return broker, nil
}
