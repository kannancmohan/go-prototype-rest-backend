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
	searchStore    store.PostSearchStore
}

func NewPostService(store store.PostDBStore, msgBrokerStore store.PostMessageBrokerStore, searchStore store.PostSearchStore) *postService {
	return &postService{store: store, msgBrokerStore: msgBrokerStore, searchStore: searchStore}
}

// Explicitly ensuring that postService adheres to the PostService interface
//var _ UserService = (*userService)(nil)

func (p *postService) GetByID(ctx context.Context, postID int64) (*model.Post, error) {
	return p.store.GetByID(ctx, postID)
}

func (p *postService) Create(ctx context.Context, payload dto.CreatePostReq) (*model.Post, error) {
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

func (p *postService) Update(ctx context.Context, payload dto.UpdatePostReq) (*model.Post, error) {
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

func (p *postService) Delete(ctx context.Context, postID int64) error {
	if err := p.store.Delete(ctx, postID); err != nil {
		return err
	}
	if err := p.msgBrokerStore.Deleted(ctx, postID); err != nil {
		return err
	}
	return nil
}

func (p *postService) Search(ctx context.Context, req store.PostSearchReq) (store.PostSearchResp, error) {
	res, err := p.searchStore.Search(ctx, req)
	if err != nil {
		return store.PostSearchResp{}, err //TODO do we need this ?
	}
	return res, nil
}
