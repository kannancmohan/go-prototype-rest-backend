package redis_postgres

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
)

type roleStore struct {
	client     *redis.Client
	orig       store.RoleStore
	expiration time.Duration
	config     *config.ApiConfig
}

func NewRoleStore(client *redis.Client, orig store.RoleStore, cfg *config.ApiConfig) *roleStore {
	return &roleStore{client: client, orig: orig, expiration: 10 * time.Minute, config: cfg} //TODO
}

func (s *roleStore) GetByName(ctx context.Context, slug string) (*model.Role, error) {
	//TODO
	return s.orig.GetByName(ctx, slug)
}
