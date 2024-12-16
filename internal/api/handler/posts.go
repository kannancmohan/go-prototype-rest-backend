package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/adapter"
)

type PostHandler struct {
	service adapter.PostService
}

func NewPostHandler(service adapter.PostService) *PostHandler {
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
		case errors.Is(err, common.ErrNotFound):
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
