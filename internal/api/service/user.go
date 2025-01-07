package service

import (
	"context"

	api_common "github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/dto"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
)

type userService struct {
	store store.UserDBStore
}

func NewUserService(store store.UserDBStore) *userService {
	return &userService{store: store}
}

// Explicitly ensuring that userService adheres to the UserService interface
//var _ UserService = (*userService)(nil)

func (u *userService) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	return u.store.GetByID(ctx, userID)
}

func (u *userService) CreateAndInvite(ctx context.Context, payload dto.CreateUserReq) (*model.User, error) {
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
		return nil, api_common.WrapErrorf(err, api_common.ErrorCodeUnknown, "error hashing password")
	}

	if err := u.store.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (u *userService) Update(ctx context.Context, payload dto.UpdateUserReq) (*model.User, error) {
	user := &model.User{
		ID:       payload.ID,
		Username: payload.Username,
		Email:    payload.Email,
		Role: model.Role{
			Name: payload.Role,
		},
	}

	// hash the user password
	if payload.Password != "" {
		if err := user.Password.Set(payload.Password); err != nil {
			return nil, api_common.WrapErrorf(err, api_common.ErrorCodeUnknown, "error hashing password")
		}
	}

	updatedUser, err := u.store.Update(ctx, user)
	if err != nil {
		return nil, err
	}
	return updatedUser, nil
}
