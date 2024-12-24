package redis_postgres

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
)

type postStore struct {
	client     *redis.Client
	orig       store.PostStore
	expiration time.Duration
	config     *config.ApiConfig
}

func NewPostStore(client *redis.Client, orig store.PostStore, cfg *config.ApiConfig) *postStore {
	return &postStore{client: client, orig: orig, expiration: 10 * time.Minute, config: cfg} //TODO
}

func (s *postStore) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	//TODO add caching logic
	return s.orig.GetByID(ctx, id)
}
