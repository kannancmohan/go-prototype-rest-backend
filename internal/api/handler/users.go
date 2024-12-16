package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/adapter"
)

// request model
type RegisterUserPayload struct {
	Username string `json:"username" validate:"required,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=3,max=72"`
	Role     string `json:"role" validate:"required,oneof=user admin moderator"`
}

type UserHandler struct {
	service adapter.UserService
}

func NewUserHandler(service adapter.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload RegisterUserPayload
	if err := readJSON(w, r, &payload); err != nil {
		badRequestResponse(w, r, err)
		return
	}

	if err := validate.Struct(payload); err != nil {
		badRequestResponse(w, r, err)
		return
	}
	ctx := r.Context()
	u, err := h.service.CreateAndInvite(ctx, adapter.CreateUserRequest{
		Username: payload.Username,
		Email:    payload.Email,
		Password: payload.Password,
		Role:     payload.Role,
	})

	if err != nil {
		switch err {
		case common.ErrDuplicateEmail:
			badRequestResponse(w, r, err)
		case common.ErrDuplicateUsername:
			badRequestResponse(w, r, err)
		default:
			internalServerError(w, r, err)
		}
		return
	}

	if err := jsonResponse(w, http.StatusCreated, u); err != nil {
		internalServerError(w, r, err)
		return
	}

}

func (h *UserHandler) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64) //TODO
	if err != nil {
		badRequestResponse(w, r, err)
		return
	}

	user, err := h.service.GetByID(r.Context(), userID)
	if err != nil {
		switch err {
		case common.ErrNotFound:
			notFoundResponse(w, r, err)
			return
		default:
			internalServerError(w, r, err)
			return
		}
	}

	if err := jsonResponse(w, http.StatusOK, user); err != nil {
		internalServerError(w, r, err)
		return
	}
}
