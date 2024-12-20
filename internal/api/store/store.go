package store

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
)

type PostStore interface {
	GetByID(context.Context, int64) (*model.Post, error)
}

type RoleStore interface {
	GetByName(context.Context, string) (*model.Role, error)
}

type UserStore interface {
	GetByID(context.Context, int64) (*model.User, error)
	Create(context.Context, *model.User) error
}
