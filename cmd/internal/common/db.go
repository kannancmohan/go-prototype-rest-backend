package app_common

import (
	"context"
	"database/sql"
	"time"
)

type DBConfig struct {
	Addr         string
	MaxOpenConns int
	MaxIdleConns int
	MaxIdleTime  time.Duration
}

func (d *DBConfig) NewConnection() (*sql.DB, error) {
	db, err := sql.Open("postgres", d.Addr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(d.MaxOpenConns)
	db.SetMaxIdleConns(d.MaxIdleConns)
	db.SetConnMaxIdleTime(d.MaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
