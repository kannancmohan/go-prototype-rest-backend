package main

import (
	"database/sql"
	"fmt"

	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
)

func initDB(env *ApiEnvVar) (*sql.DB, error) {
	dbCfg := app_common.DBConfig{
		Addr:         fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", env.DBUser, env.DBPass, env.DBHost, env.ApiDBName, env.DBSslMode),
		MaxOpenConns: env.ApiDBMaxOpenConns,
		MaxIdleConns: env.ApiDBMaxIdleConns,
		MaxIdleTime:  env.ApiDBMaxIdleTime,
	}
	db, err := dbCfg.NewConnection()
	if err != nil {
		return nil, err
	}
	return db, nil
}
