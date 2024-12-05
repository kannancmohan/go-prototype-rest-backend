package handler

import "github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"

type PostHandler struct {
	store store.PostStore
}

func NewPostHandler(store store.PostStore) *PostHandler {
	return &PostHandler{store: store}
}
