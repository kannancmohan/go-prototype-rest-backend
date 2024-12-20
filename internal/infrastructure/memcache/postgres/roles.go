package postgres_memcache

import (
	"context"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
)

type roleStore struct {
	client     *memcache.Client
	orig       store.RoleStore
	expiration time.Duration
	config     *config.ApiConfig
}

func NewRoleStore(client *memcache.Client, orig store.RoleStore, cfg *config.ApiConfig) *roleStore {
	return &roleStore{client: client, orig: orig, expiration: 10 * time.Minute, config: cfg} //TODO
}

func (s *roleStore) GetByName(ctx context.Context, slug string) (*model.Role, error) {
	//TODO
	return s.orig.GetByName(ctx, slug)
}
