package app_common

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addr string
	DB   int
}

func (d *RedisConfig) NewConnection() (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: d.Addr,
		DB:   d.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, err
	}
	return rdb, nil
}
