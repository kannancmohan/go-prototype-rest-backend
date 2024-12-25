package redis_postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
	"github.com/redis/go-redis/v9"
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
	cacheKey := userCacheKey(userID)
	user := &model.User{}

	// Attempt to fetch user from Redis
	cacheData, err := s.client.Get(ctx, cacheKey).Result()
	if err == nil { // Cache hit
		err = json.Unmarshal([]byte(cacheData), user)
		if err != nil {
			return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to unmarshal user from cache")
		}
		fmt.Println("cache hit....")
		return user, nil
	} else if err != redis.Nil { // Redis error (other than a cache miss)
		return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "redis get failed")
	}
	return s.orig.GetByID(ctx, userID)
}

func (s *userStore) Create(ctx context.Context, user *model.User) error {
	if err := s.orig.Create(ctx, user); err != nil {
		return err
	}
	cacheKey := userCacheKey(user.ID)
	userJSON, err := json.Marshal(user)
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to marshal user")
	}
	err = s.client.Set(ctx, cacheKey, userJSON, s.config.RedisCacheTTL).Err()
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to set user cache")
	}
	return nil
}

func userCacheKey(userID int64) string {
	return fmt.Sprintf("user:%d", userID)
}
