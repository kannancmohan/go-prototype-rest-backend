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
	postID, err := strconv.ParseInt(chi.URLParam(r, "postID"), 10, 64) //TODO
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}

	post, err := h.service.GetByID(r.Context(), postID)
	if err != nil {
		renderErrorResponse(w, "create failed", err)
		return
	}

	renderResponse(w, http.StatusCreated, post)
}
