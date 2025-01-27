package main

import (
	"fmt"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
)

type EnvVar struct {
	LogLevel           string
	KafkaHost          string
	KafkaConsumerTopic string
	ElasticHost        string
	ElasticIndexName   string
}

// string representation to hide sensitive fields.
func (e EnvVar) String() string {
	return fmt.Sprintf("EnvVar{LogLevel: %s, KafkaHost: %s, KafkaConsumerTopic: %s}", e.LogLevel, e.KafkaHost, e.KafkaConsumerTopic)
}

func initEnvVar(sec store.SecretFetchStore) *EnvVar {
	return &EnvVar{
		LogLevel:           sec.GetEnvString("LOG_LEVEL", "info"), // supported values DEBUG,INFO,WARN,ERROR
		KafkaHost:          sec.GetEnvString("KAFKA_HOST", "192.168.0.30:9093"),
		KafkaConsumerTopic: sec.GetEnvString("API_KAFKA_TOPIC", "posts"),
		ElasticHost:        sec.GetEnvString("ELASTIC_HOST", "http://192.168.0.30:9200"),
		ElasticIndexName:   sec.GetEnvString("ELASTIC_POST_INDEX_NAME", "posts"),
	}
}
