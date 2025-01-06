package main

import (
	"fmt"

	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
)

type EnvVar struct {
	AppAddr            string
	LogLevel           string
	KafkaHost          string
	KafkaConsumerTopic string
	ElasticHost        string
	ElasticIndexName   string
}

// string representation to hide sensitive fields.
func (e EnvVar) String() string {
	return fmt.Sprintf("EnvVar{ApiAddr: %s, LogLevel: %s, KafkaHost: %s, KafkaConsumerTopic: %s}", e.AppAddr, e.LogLevel, e.KafkaHost, e.KafkaConsumerTopic)
}

func initEnvVar(envName string) *EnvVar {
	env := app_common.NewEnvVarFetcher(envName, nil)
	return &EnvVar{
		AppAddr:            fmt.Sprintf(":%s", env.GetEnvString("PORT", "8080")),
		LogLevel:           env.GetEnvString("LOG_LEVEL", "info"), // supported values DEBUG,INFO,WARN,ERROR
		KafkaHost:          env.GetEnvString("KAFKA_HOST", "192.168.0.30:9093"),
		KafkaConsumerTopic: env.GetEnvString("API_KAFKA_TOPIC", "posts"),
		ElasticHost:        env.GetEnvString("ELASTIC_HOST", "http://192.168.0.30:9200"),
		ElasticIndexName:   env.GetEnvString("ELASTIC_POST_INDEX_NAME", "posts"),
	}
}
