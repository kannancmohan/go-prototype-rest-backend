package redis_postgres

import (
	"context"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
	"github.com/redis/go-redis/v9"
)

type roleStore struct {
	client     *redis.Client
	orig       store.RoleStore
	expiration time.Duration
}

func NewRoleStore(client *redis.Client, orig store.RoleStore, expiration time.Duration) *roleStore {
	return &roleStore{client: client, orig: orig, expiration: expiration} //TODO
}

func (s *roleStore) GetByName(ctx context.Context, slug string) (*model.Role, error) {
	//TODO
	return s.orig.GetByName(ctx, slug)
}
