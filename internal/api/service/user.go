package service

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
)

type UserService interface {
	GetByID(context.Context, int64) (*model.User, error)
}

type userService struct {
	store store.UserStore
}

// Explicitly ensuring that userService adheres to the UserService interface
var _ UserService = (*userService)(nil)

func (u *userService) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	return u.store.GetByID(ctx, userID)
}
