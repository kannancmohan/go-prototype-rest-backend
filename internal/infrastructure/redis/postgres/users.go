package redis_postgres

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
)

type userStore struct {
	client     *redis.Client
	orig       store.UserStore
	expiration time.Duration
	config     *config.ApiConfig
}

func NewUserStore(client *redis.Client, orig store.UserStore, cfg *config.ApiConfig) *userStore {
	return &userStore{client: client, orig: orig, expiration: 10 * time.Minute, config: cfg}
}

func (s *userStore) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	//TODO add cache logic
	return s.orig.GetByID(ctx, userID)
}

func (s *userStore) Create(ctx context.Context, user *model.User) error {
	//TODO set the new user to cache
	return nil
}
