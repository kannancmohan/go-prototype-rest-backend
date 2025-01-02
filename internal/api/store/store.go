package store

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
)

// store interface for persisting user,role & post. In this case, to postgres db
type PostStore interface {
	GetByID(context.Context, int64) (*model.Post, error)
	Create(context.Context, *model.Post) error
	Update(context.Context, *model.Post) (*model.Post, error)
}

type RoleStore interface {
	GetByName(context.Context, string) (*model.Role, error)
}

type UserStore interface {
	GetByID(context.Context, int64) (*model.User, error)
	Create(context.Context, *model.User) error
	Update(context.Context, *model.User) (*model.User, error)
}

// store interface for post broker events. In this case, to kafka
type PostMessageBrokerStore interface {
	Created(ctx context.Context, post model.Post) error
	Deleted(ctx context.Context, id int64) error
	Updated(ctx context.Context, post model.Post) error
}
