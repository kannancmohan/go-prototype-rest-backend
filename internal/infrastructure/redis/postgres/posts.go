package redis_postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
	"github.com/redis/go-redis/v9"
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
	cacheKey := postCacheKey(id)
	post := &model.Post{}

	// Attempt to fetch user from Redis
	cacheData, err := s.client.Get(ctx, cacheKey).Result()
	if err == nil { // Cache hit
		err = json.Unmarshal([]byte(cacheData), post)
		if err != nil {
			return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to unmarshal post from cache")
		}
		slog.Debug("retrieved post from cache", "postID", id)
		return post, nil
	} else if err != redis.Nil { // Redis error (other than a cache miss)
		return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "redis get failed")
	}
	p, err := s.orig.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.cachePost(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *postStore) Create(ctx context.Context, post *model.Post) error {
	//TODO add transaction for atomicity ??
	if err := s.orig.Create(ctx, post); err != nil {
		return err
	}

	if err := s.cachePost(ctx, post); err != nil {
		return err
	}
	return nil
}

func (s *postStore) Update(ctx context.Context, post *model.Post) (*model.Post, error) {
	//TODO add transaction for atomicity ??
	updatedPost, err := s.orig.Update(ctx, post)
	if err != nil {
		return nil, err
	}
	if err := s.cachePost(ctx, updatedPost); err != nil {
		return nil, err
	}
	return updatedPost, nil
}

func postCacheKey(value any) string {
	return fmt.Sprintf("post:%v", value)
}

func (s *postStore) cachePost(ctx context.Context, post *model.Post) error {
	postJSON, err := json.Marshal(post)
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "failed to marshal post")
	}
	cacheKey := postCacheKey(post.ID)
	s.client.Set(ctx, cacheKey, postJSON, s.config.RedisCacheTTL)
	return nil
}
