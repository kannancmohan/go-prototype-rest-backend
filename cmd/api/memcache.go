package main

import (
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

func initMemcached(env *ApiEnvVar) (*memcache.Client, error) {
	client := memcache.New(env.MemCacheDHost)
	if err := client.Ping(); err != nil {
		return nil, err
	}
	client.Timeout = 100 * time.Millisecond
	client.MaxIdleConns = 100
	return client, nil
}
