package service

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
)

type CreateUserRequest struct {
	Username string
	Email    string
	Password string
	Role     string
}

type UserService interface {
	GetByID(context.Context, int64) (*model.User, error)
	CreateAndInvite(context.Context, CreateUserRequest) (*model.User, error)
}

type userService struct {
	store store.UserStore
}

// Explicitly ensuring that userService adheres to the UserService interface
var _ UserService = (*userService)(nil)

func (u *userService) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	return u.store.GetByID(ctx, userID)
}

func (u *userService) CreateAndInvite(ctx context.Context, payload CreateUserRequest) (*model.User, error) {
	user := &model.User{
		Username: payload.Username,
		Email:    payload.Email,
		Role: model.Role{
			Name: payload.Role,
		},
	}

	// hash the user password
	if err := user.Password.Set(payload.Password); err != nil {
		//app.internalServerError(w, r, err)
		return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "error hashing password")
	}
	//return u.store.Create(ctx)
	return nil, nil
}
