package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/service"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/store"
)

type PostHandler struct {
	service service.PostService
}

func NewPostHandler(service service.PostService) *PostHandler {
	return &PostHandler{service: service}
}

func (h *PostHandler) GetPostHandler(w http.ResponseWriter, r *http.Request) {
	//post := getPostFromCtx(r)

	// comments, err := h.service.GetByID(r.Context(), post.ID)
	// if err != nil {
	// 	internalServerError(w, r, err)
	// 	return
	// }

	// post.Comments = comments
	postID, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64) //TODO
	if err != nil {
		badRequestResponse(w, r, err)
		return
	}
	post, err := h.service.GetByID(r.Context(), postID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			notFoundResponse(w, r, err)
		default:
			internalServerError(w, r, err)
		}
		return
	}

	if err := jsonResponse(w, http.StatusOK, post); err != nil {
		internalServerError(w, r, err)
		return
	}
}
