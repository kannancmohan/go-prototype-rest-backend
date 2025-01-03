package main

import (
	//"log"
	"log/slog"
	"os"

	esv8 "github.com/elastic/go-elasticsearch/v8"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
)

func main() {
	envName := app_common.GetEnvNameFromCommandLine()
	env := initEnvVar(envName)
	initLogger(env) // error ignored on purpose
	// es, err := initElasticSearch(env)
	// if err != nil {
	// 	log.Fatalf("Error init ElasticSearch: %s", err)
	// }
	// kafka, err := initKafkaConsumer(env)
	// if err != nil {
	// 	log.Fatalf("Error init KafkaConsumer: %s", err)
	// }

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
