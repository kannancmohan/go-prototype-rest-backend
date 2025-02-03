package main

import (
	"log"

	api_app "github.com/kannancmohan/go-prototype-rest-backend/cmd/api/app"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
)

func main() {
	err := api_app.ListenAndServe(app_common.GetEnvNameFromCommandLine(), app_common.SysInterruptStopChan())
	if err != nil {
		log.Fatalf("Error starting aoi app: %s", err.Error())
	}
}
