package main

import (
	"context"
	"log"

	indexer_app "github.com/kannancmohan/go-prototype-rest-backend/cmd/elasticsearch-indexer-kafka/app"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
)

func main() {

	err := indexer_app.ListenAndServe(context.Background(), app_common.GetEnvNameFromCommandLine(), app_common.SysInterruptStopChan())
	if err != nil {
		log.Fatalf("Error starting indexer app: %s", err.Error())
	}
}
