package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
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

	cacheData, err := s.client.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		slog.Debug("user not found in cache", "userID", userID)
	} else if err != nil {
		return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "redis get (user) failed")
	} else {
		// Cache hit: Unmarshal user JSON
		var user model.User
		if jsonErr := json.Unmarshal([]byte(cacheData), &user); jsonErr != nil {
			return nil, common.WrapErrorf(jsonErr, common.ErrorCodeUnknown, "failed to unmarshal user from cache")
		}
		slog.Debug("retrieved user from cache", "userID", userID)
		return &user, nil
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

func (s *userStore) GetByEmail(ctx context.Context, userEmail string) (*model.User, error) {
	cacheEmailKey := userCacheKey(strings.ToLower(userEmail))

	// Attempt to fetch user by email from Redis
	userIDStr, err := s.client.Get(ctx, cacheEmailKey).Result()
	if err == redis.Nil {
		slog.Debug("email not found in cache", "email", userEmail)
	} else if err != nil {
		return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "redis get (email) failed")
	} else {
		if userID, convErr := strconv.ParseInt(userIDStr, 10, 64); convErr == nil {
			return s.GetByID(ctx, userID)
		}
	}

	u, err := s.orig.GetByEmail(ctx, userEmail)
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

func (s *userStore) Delete(ctx context.Context, userID int64) error {
	//TODO add transaction for atomicity ??
	err := s.orig.Delete(ctx, userID)
	if err != nil {
		return err
	}
	cacheKey := userCacheKey(userID)
	if err := s.client.Del(ctx, cacheKey).Err(); err != nil {
		return err
	}
	return nil
}

func userCacheKey(value any) string {
	return fmt.Sprintf("user:%v", value)
}

func (s *userStore) cacheUser(ctx context.Context, user *model.User) error {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to marshal user")
	}
	cacheKey := userCacheKey(user.ID)
	emailCacheKey := userCacheKey(strings.ToLower(user.Email))
	// Use Redis pipeline with a transaction
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, cacheKey, userJSON, s.expiration)     // cache user
	pipe.Set(ctx, emailCacheKey, user.ID, s.expiration) // cache user:email
	_, err = pipe.Exec(ctx)                             // Atomically executes all commands in the pipeline
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to cache user update")
	}
	return nil
}
