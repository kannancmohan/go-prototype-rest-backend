package main

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/env"
)

func NewMemcached(env *env.EnvVar) (*memcache.Client, error) {
	client := memcache.New(env.MemCacheDHost)
	if err := client.Ping(); err != nil {
		return nil, err
	}
	client.Timeout = 100 * time.Millisecond
	client.MaxIdleConns = 100
	return client, nil
}
