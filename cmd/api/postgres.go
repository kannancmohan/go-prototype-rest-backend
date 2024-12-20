package main

import (
	"database/sql"
	"fmt"
	common_db "github.com/kannancmohan/go-prototype-rest-backend/internal/common/db"
)

func initDB(env *EnvVar) (*sql.DB, error) {
	dbCfg := common_db.DBConfig{
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
