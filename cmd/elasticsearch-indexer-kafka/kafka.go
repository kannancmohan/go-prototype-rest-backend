package main

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
)

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
