package handler

import (
	"net/http"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/dto"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
)

type CreatePostPayload struct {
	UserID  int64    `json:"user_id"` //TODO update to set this dynamically instead of passing as part of json request
	Title   string   `json:"title" validate:"required,max=100"`
	Content string   `json:"content" validate:"required,max=1000"`
	Tags    []string `json:"tags"`
}

type UpdatePostPayload struct {
	Title   string   `json:"title" validate:"omitempty,max=100"`
	Content string   `json:"content" validate:"omitempty,max=1000"`
	Tags    []string `json:"tags" validate:"omitempty"`
}

type SearchPostPayload struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	UserID  int64    `json:"user_id"`
	Tags    []string `json:"tags"`
}

type PostHandler struct {
	service PostService
}

func NewPostHandler(service PostService) *PostHandler {
	return &PostHandler{service: service}
}

func (h *PostHandler) GetPostHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *PostHandler) CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := readJSONValid[CreatePostPayload](w, r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}
	ctx := r.Context()
	u, err := h.service.Create(ctx, dto.CreatePostReq{
		UserID:  payload.UserID, //TODO update to set this dynamically instead of passing as part of json request
		Content: payload.Content,
		Title:   payload.Title,
		Tags:    payload.Tags,
	})

	if err != nil {
		renderErrorResponse(w, "create failed", err)
		return
	}
	renderResponse(w, http.StatusCreated, u)
}

func (h *PostHandler) UpdatePostHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIntParam("postID", r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}
	payload, err := readJSONValid[UpdatePostPayload](w, r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}
	ctx := r.Context()
	u, err := h.service.Update(ctx, dto.UpdatePostReq{
		ID:      id,
		Title:   payload.Title,
		Content: payload.Content,
		Tags:    payload.Tags,
	})

	if err != nil {
		renderErrorResponse(w, "update failed", err)
		return
	}
	renderResponse(w, http.StatusOK, u)
}

func (h *PostHandler) DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIntParam("postID", r)
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

func (h *PostHandler) SearchPostHandler(w http.ResponseWriter, r *http.Request) {
	payload, err := readJSONValid[SearchPostPayload](w, r)
	if err != nil {
		renderErrorResponse(w, "invalid request", err)
		return
	}
	ctx := r.Context()
	res, err := h.service.Search(ctx, store.PostSearchReq{
		Title:   payload.Title,
		Content: payload.Content,
		UserID:  payload.UserID,
		Tags:    payload.Tags,
	})
	if err != nil {
		renderErrorResponse(w, "search post failed", err)
		return
	}
	renderResponse(w, http.StatusOK, res)
}
