package app_common

import (
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaProducerConfig struct {
	Addr string
}

func (d *KafkaProducerConfig) NewKafkaProducer() (*kafka.Producer, error) {
	config := kafka.ConfigMap{
		"bootstrap.servers": d.Addr,
	}

	client, err := kafka.NewProducer(&config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

type KafkaConsumerConfig struct {
	Addr             string
	GroupID          string
	AutoOffsetRest   string
	EnableAutoCommit bool
	Topics           []string
}

func (d *KafkaConsumerConfig) NewKafkaConsumer() (*kafka.Consumer, error) {
	config := kafka.ConfigMap{
		"bootstrap.servers":  d.Addr,
		"group.id":           d.GroupID,
		"auto.offset.reset":  d.AutoOffsetRest,
		"enable.auto.commit": d.EnableAutoCommit,
	}

	client, err := kafka.NewConsumer(&config)
	if err != nil {
		return nil, err
	}
	slog.Debug("kafka consumer created", "groupID", d.GroupID)

	if err := client.SubscribeTopics(d.Topics, nil); err != nil {
		return nil, err
	}
	slog.Debug("kafka consumer subscribed to topic", "topic", d.Topics)

	return client, nil

}
