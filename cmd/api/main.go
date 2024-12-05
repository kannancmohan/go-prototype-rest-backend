package main

import (
	"log"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/db"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/env"
)

func main() {

	dbCfg := db.DBConfig{
		Addr:         env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost/socialnetwork?sslmode=disable"),
		MaxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
		MaxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
		MaxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
	}
	db, err := dbCfg.NewConnection()
	if err != nil {
		log.Fatal(err)
	}

	app := api.NewAPI(db)
	log.Fatal(app.Run())
}
