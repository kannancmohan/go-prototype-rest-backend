package service

import (
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/adapter"
)

func NewService(store adapter.Storage) adapter.Service {
	return adapter.Service{
		UserService: &userService{store: store.Users},
		PostService: &postService{store: store.Posts},
	}
}
