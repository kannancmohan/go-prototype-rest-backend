package main

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
)

func initKafkaProducer(env *ApiEnvVar) (*kafka.Producer, error) {
	kafkaProd := app_common.KafkaProducerConfig{
		Addr: env.KafkaHost,
	}
	p, err := kafkaProd.NewKafkaProducer()
	if err != nil {
		return nil, err
	}
	return p, nil
}
