package main

import (
	indexer_app "github.com/kannancmohan/go-prototype-rest-backend/cmd/elasticsearch-indexer-kafka/app"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
	"log"
)

func main() {

	err := indexer_app.ListenAndServe(app_common.GetEnvNameFromCommandLine(), app_common.SysInterruptStopChan())
	if err != nil {
		log.Fatalf("Error starting indexer app: %s", err.Error())
	}
}
