package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type PostHandler struct {
	service PostService
}

func NewPostHandler(service PostService) *PostHandler {
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
	id, err := getIntParam("postID", r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}

	post, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		renderErrorResponse(w, "find failed", err)
		return
	}

	renderResponse(w, http.StatusCreated, post)
}
