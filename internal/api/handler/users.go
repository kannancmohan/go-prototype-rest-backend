package handler

import (
	"net/http"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/dto"
)

// request model
type RegisterUserPayload struct {
	Username string `json:"username" validate:"required,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=3,max=72"`
	Role     string `json:"role" validate:"required,oneof=user admin moderator"`
}

type UserHandler struct {
	service UserService
}

func NewUserHandler(service UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := readJSONValid[RegisterUserPayload](w, r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}
	ctx := r.Context()
	u, err := h.service.CreateAndInvite(ctx, dto.CreateUserRequest{
		Username: payload.Username,
		Email:    payload.Email,
		Password: payload.Password,
		Role:     payload.Role,
	})

	if err != nil {
		renderErrorResponse(w, "create failed", err)
		return
	}
	renderResponse(w, http.StatusCreated, u)
}

func (h *UserHandler) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIntParam("userID", r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}

	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		renderErrorResponse(w, "find failed", err)
		return
	}

	renderResponse(w, http.StatusOK, user)
}
