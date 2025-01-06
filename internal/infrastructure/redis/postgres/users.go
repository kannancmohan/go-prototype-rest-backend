package redis_postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	api_common "github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
	"github.com/redis/go-redis/v9"
)

type userStore struct {
	client     *redis.Client
	orig       store.UserDBStore
	expiration time.Duration
}

func NewUserStore(client *redis.Client, orig store.UserDBStore, expiration time.Duration) *userStore {
	return &userStore{client: client, orig: orig, expiration: expiration}
}

func (s *userStore) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	cacheKey := userCacheKey(userID)
	user := &model.User{}

	// Attempt to fetch user from Redis
	cacheData, err := s.client.Get(ctx, cacheKey).Result()
	if err == nil { // Cache hit
		err = json.Unmarshal([]byte(cacheData), user)
		if err != nil {
			return nil, api_common.WrapErrorf(err, api_common.ErrorCodeUnknown, "failed to unmarshal user from cache")
		}
		slog.Debug("retrieved user from cache", "userID", userID)
		return user, nil
	} else if err != redis.Nil { // Redis error (other than a cache miss)
		return nil, api_common.WrapErrorf(err, api_common.ErrorCodeUnknown, "redis get failed")
	}
	u, err := s.orig.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.cacheUser(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *userStore) Create(ctx context.Context, user *model.User) error {
	//TODO add transaction for atomicity ??
	if err := s.orig.Create(ctx, user); err != nil {
		return err
	}
	if err := s.cacheUser(ctx, user); err != nil {
		return err
	}

	return nil
}

func (s *userStore) Update(ctx context.Context, user *model.User) (*model.User, error) {
	//TODO add transaction for atomicity ??
	u, err := s.orig.Update(ctx, user)
	if err != nil {
		return nil, err
	}
	if err := s.cacheUser(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func userCacheKey(value any) string {
	return fmt.Sprintf("user:%v", value)
}

func (s *userStore) cacheUser(ctx context.Context, user *model.User) error {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return api_common.WrapErrorf(err, api_common.ErrorCodeUnknown, "failed to marshal user")
	}
	cacheKey := userCacheKey(user.ID)
	emailCacheKey := userCacheKey(user.Email)
	// Use Redis pipeline with a transaction
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, cacheKey, userJSON, s.expiration)     // cache user
	pipe.Set(ctx, emailCacheKey, user.ID, s.expiration) // cache user:email
	_, err = pipe.Exec(ctx)                             // Atomically executes all commands in the pipeline
	if err != nil {
		return api_common.WrapErrorf(err, api_common.ErrorCodeUnknown, "failed to cache user update")
	}
	return nil
}
