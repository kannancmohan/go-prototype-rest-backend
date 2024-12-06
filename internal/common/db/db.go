package db

import (
	"context"
	"database/sql"
	"time"
)

type DBConfig struct {
	Addr         string
	MaxOpenConns int
	MaxIdleConns int
	MaxIdleTime  string
}

func (d *DBConfig) NewConnection() (*sql.DB, error) {
	db, err := sql.Open("postgres", d.Addr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(d.MaxOpenConns)
	db.SetMaxIdleConns(d.MaxIdleConns)

	duration, err := time.ParseDuration(d.MaxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

// func NewConnection(addr string, maxOpenConns, maxIdleConns int, maxIdleTime string) (*sql.DB, error) {
// 	db, err := sql.Open("postgres", addr)
// 	if err != nil {
// 		return nil, err
// 	}

// 	db.SetMaxOpenConns(maxOpenConns)
// 	db.SetMaxIdleConns(maxIdleConns)

// 	duration, err := time.ParseDuration(maxIdleTime)
// 	if err != nil {
// 		return nil, err
// 	}
// 	db.SetConnMaxIdleTime(duration)

// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	if err = db.PingContext(ctx); err != nil {
// 		return nil, err
// 	}

// 	return db, nil
// }