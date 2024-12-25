package main

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func initRedis(env *ApiEnvVar) (*redis.Client, error) {
	host := env.RedisHost
	db := env.RedisDB
	dbi, _ := strconv.Atoi(db)
	rdb := redis.NewClient(&redis.Options{
		Addr: host,
		DB:   dbi,
	})
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		return nil, err
	}
	return rdb, nil
}
