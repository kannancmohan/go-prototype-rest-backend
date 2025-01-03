package service

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/dto"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
)

type postService struct {
	store          store.PostDBStore
	msgBrokerStore store.PostMessageBrokerStore
}

func NewPostService(store store.PostDBStore, msgBrokerStore store.PostMessageBrokerStore) *postService {
	return &postService{store: store, msgBrokerStore: msgBrokerStore}
}

// Explicitly ensuring that postService adheres to the PostService interface
//var _ UserService = (*userService)(nil)

func (p *postService) GetByID(ctx context.Context, postID int64) (*model.Post, error) {
	return p.store.GetByID(ctx, postID)
}

func (p *postService) Create(ctx context.Context, payload dto.CreatePostRequest) (*model.Post, error) {
	post := &model.Post{
		Content: payload.Content,
		Title:   payload.Title,
		UserID:  payload.UserID,
		Tags:    payload.Tags,
	}

	if err := p.store.Create(ctx, post); err != nil {
		return nil, err
	}

	if err := p.msgBrokerStore.Created(ctx, *post); err != nil {
		return nil, err
	}
	return post, nil
}

func (p *postService) Update(ctx context.Context, payload dto.UpdatePostRequest) (*model.Post, error) {
	post := &model.Post{
		ID:      payload.ID,
		Content: payload.Content,
		Title:   payload.Title,
		Tags:    payload.Tags,
	}

	updatedPost, err := p.store.Update(ctx, post)
	if err != nil {
		return nil, err
	}
	if err := p.msgBrokerStore.Updated(ctx, *updatedPost); err != nil {
		return nil, err
	}
	return updatedPost, nil
}
