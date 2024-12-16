package adapter

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
)

type UserService interface {
	GetByID(context.Context, int64) (*model.User, error)
	CreateAndInvite(context.Context, CreateUserRequest) (*model.User, error)
}

type PostService interface {
	GetByID(context.Context, int64) (*model.Post, error)
}

type Service struct {
	UserService UserService
	PostService PostService
}
