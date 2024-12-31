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
		return user, nil
	} else if err != redis.Nil { // Redis error (other than a cache miss)
		return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "redis get failed")
	}
	return s.orig.GetByID(ctx, userID)
}

func (s *userStore) Create(ctx context.Context, user *model.User) error {
	//TODO add transaction for atomicity ??
	if err := s.orig.Create(ctx, user); err != nil {
		return err
	}
	userJSON, err := json.Marshal(user)
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to marshal user")
	}

	cacheKey := userCacheKey(user.ID)
	emailCacheKey := userCacheKey(user.Email)
	// Use Redis pipeline with a transaction
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, cacheKey, userJSON, s.config.RedisCacheTTL)     // cache user
	pipe.Set(ctx, emailCacheKey, user.ID, s.config.RedisCacheTTL) // cache user:email
	_, err = pipe.Exec(ctx)                                       // Atomically executes all commands in the pipeline
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to cache user create")
	}

	return nil
}

func (s *userStore) Update(ctx context.Context, user *model.User) (*model.User, error) {
	//TODO add transaction for atomicity ??
	u, err := s.orig.Update(ctx, user)
	if err != nil {
		return nil, err
	}
	userJSON, err := json.Marshal(u)
	if err != nil {
		return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to marshal user")
	}

	cacheKey := userCacheKey(u.ID)
	emailCacheKey := userCacheKey(u.Email)
	// Use Redis pipeline with a transaction
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, cacheKey, userJSON, s.config.RedisCacheTTL)  // cache user
	pipe.Set(ctx, emailCacheKey, u.ID, s.config.RedisCacheTTL) // cache user:email
	_, err = pipe.Exec(ctx)                                    // Atomically executes all commands in the pipeline
	if err != nil {
		return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to cache user update")
	}

	return u, nil
}

func userCacheKey(value any) string {
	return fmt.Sprintf("user:%v", value)
}
