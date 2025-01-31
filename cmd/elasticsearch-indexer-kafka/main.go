package main

import (
	"github.com/kannancmohan/go-prototype-rest-backend/cmd/elasticsearch-indexer-kafka/app"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
	"log"
)

func main() {

	if err := app.StartApp(app_common.GetEnvNameFromCommandLine()); err != nil {
		log.Fatalf("Error starting application: %s", err.Error())
	}
}
