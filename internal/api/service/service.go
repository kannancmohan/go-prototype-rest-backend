package service

import "github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"

type Service struct {
	UserService UserService
	PostService PostService
}

func NewService(store store.Storage) Service {
	return Service{
		UserService: &userService{store: store.Users},
		PostService: &postService{store: store.Posts},
	}
}
