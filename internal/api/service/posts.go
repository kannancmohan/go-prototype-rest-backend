package service

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
)

type PostService interface {
	GetByID(context.Context, int64) (*model.Post, error)
}

type postService struct {
	store store.PostStore
}

// Explicitly ensuring that postService adheres to the PostService interface
var _ UserService = (*userService)(nil)

func (p *postService) GetByID(ctx context.Context, postID int64) (*model.Post, error) {
	return p.store.GetByID(ctx, postID)
}
