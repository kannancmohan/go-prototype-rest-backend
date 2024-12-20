package service

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
)

type postService struct {
	store store.PostStore
}

func NewPostService(store store.PostStore) *postService {
	return &postService{store: store}
}

// Explicitly ensuring that postService adheres to the PostService interface
//var _ UserService = (*userService)(nil)

func (p *postService) GetByID(ctx context.Context, postID int64) (*model.Post, error) {
	return p.store.GetByID(ctx, postID)
}
