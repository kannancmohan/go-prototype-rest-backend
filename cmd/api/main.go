package main

import (
	"log"

	api_app "github.com/kannancmohan/go-prototype-rest-backend/cmd/api/app"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
)

func main() {
	appServer, err := api_app.NewAppServer(app_common.GetEnvNameFromCommandLine())

	if err != nil {
		log.Fatalf("Error initializing app server: %s", err.Error())
	}

	appServer.ListenForStopChannels(app_common.SysInterruptStopChan())
	if err := appServer.Start(); err != nil {
		log.Fatalf("Server error: %s", err.Error())
	}

}
