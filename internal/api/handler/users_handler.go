package handler

import (
	"net/http"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/dto"
)

type registerUserPayload struct {
	Username string `json:"username" validate:"required,max=100"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=3,max=72"`
	Role     string `json:"role" validate:"required,oneof=user admin moderator"`
}

type updateUserPayload struct {
	Username string `json:"username" validate:"omitempty,max=100"`
	Email    string `json:"email" validate:"omitempty,email,max=255"`
	Password string `json:"password" validate:"omitempty,min=3,max=72"`
	Role     string `json:"role" validate:"omitempty,oneof=user admin moderator"`
}

type UserHandler struct {
	service UserService
}

func NewUserHandler(service UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) RegisterUserHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := readJSONValid[registerUserPayload](w, r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}
	ctx := r.Context()
	u, err := h.service.CreateAndInvite(ctx, dto.CreateUserReq{
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

func (h *UserHandler) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getPathVariableAsInt("userID", r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}

	payload, err := readJSONValid[updateUserPayload](w, r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}
	ctx := r.Context()
	u, err := h.service.Update(ctx, dto.UpdateUserReq{
		ID:       id,
		Username: payload.Username,
		Email:    payload.Email,
		Password: payload.Password,
		Role:     payload.Role,
	})

	if err != nil {
		renderErrorResponse(w, "update failed", err)
		return
	}
	renderResponse(w, http.StatusOK, u)
}

func (h *UserHandler) GetUserByIdHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getPathVariableAsInt("userID", r)
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

func (h *UserHandler) GetUserByEmailHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := getPathVariableAsStr("userEmail", r)

	user, err := h.service.GetByEmail(r.Context(), userEmail)
	if err != nil {
		renderErrorResponse(w, "find failed", err)
		return
	}

	renderResponse(w, http.StatusOK, user)
}

func (h *UserHandler) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getPathVariableAsInt("userID", r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}
	ctx := r.Context()
	if err := h.service.Delete(ctx, id); err != nil {
		renderErrorResponse(w, "delete failed", err)
		return
	}
	renderResponse(w, http.StatusOK, id)
}
