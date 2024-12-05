package handler

import (
	"net/http"
	"strconv"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
)

type UserHandler struct {
	store store.UserStore
}

func NewUserHandler(store store.UserStore) *UserHandler {
	return &UserHandler{store: store}
}

func (handler *UserHandler) getUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		badRequestResponse(w, r, err)
		return
	}

	user, err := handler.store.GetByID(r.Context(), userID)
	if err != nil {
		switch err {
		case store.ErrNotFound:
			notFoundResponse(w, r, err)
			return
		default:
			internalServerError(w, r, err)
			return
		}
	}

	if err := jsonResponse(w, http.StatusOK, user); err != nil {
		internalServerError(w, r, err)
	}
}
