package main

import (
	"database/sql"
	"fmt"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/db"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/env"
)

func initDB(env *env.EnvVar) (*sql.DB, error) {
	dbCfg := db.DBConfig{
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
