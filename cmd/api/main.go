package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/service"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
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

	conf := api.ApiConfig{
		Addr:              env.GetString("ADDR", ":8080"),
		CorsAllowedOrigin: env.GetString("CORS_ALLOWED_ORIGIN", "http://localhost:8080"),
	}
	s, err := newServer(conf, db)
	log.Fatal(runSever(s))
}

func newServer(config api.ApiConfig, db *sql.DB) (*http.Server, error) {
	pStore := store.NewPostStore(db)
	//rStore := store.NewRoleStore(db)
	uStore := store.NewUserStore(db)

	pService := service.NewPostService(pStore)
	uService := service.NewUserService(uStore)

	handler := handler.NewHandler(uService, pService)

	router := api.NewRouter(handler, config)
	routes := router.RegisterHandlers()
	return &http.Server{
		Addr:         config.Addr,
		Handler:      routes,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}, nil
}

func runSever(s *http.Server) error {
	err := s.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
