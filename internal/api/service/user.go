package service

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/dto"
)

type userService struct {
	store UserStore
}

func NewUserService(store UserStore) *userService {
	return &userService{store: store}
}

// Explicitly ensuring that userService adheres to the UserService interface
//var _ UserService = (*userService)(nil)

func (u *userService) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	return u.store.GetByID(ctx, userID)
}

func (u *userService) CreateAndInvite(ctx context.Context, payload dto.CreateUserRequest) (*model.User, error) {
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

	if err := u.store.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
