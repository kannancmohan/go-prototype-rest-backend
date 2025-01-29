package store

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
)

//go:generate mockgen -destination=mocks/mock_db.go -package=mockstore github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store PostDBStore,RoleDBStore,UserDBStore

// store interface for persisting user,role & post. In this case, to postgres db
type PostDBStore interface {
	GetByID(context.Context, int64) (*model.Post, error)
	Create(context.Context, *model.Post) error
	Update(context.Context, *model.Post) (*model.Post, error)
	Delete(context.Context, int64) error
}

type RoleDBStore interface {
	GetByName(context.Context, string) (*model.Role, error)
}

type UserDBStore interface {
	GetByID(context.Context, int64) (*model.User, error)
	Create(context.Context, *model.User) error
	Update(context.Context, *model.User) (*model.User, error)
	Delete(context.Context, int64) error
}
