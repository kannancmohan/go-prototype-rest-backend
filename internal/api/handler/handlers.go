package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/adapter"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

type UserService interface {
	GetByID(context.Context, int64) (*model.User, error)
	CreateAndInvite(context.Context, adapter.CreateUserRequest) (*model.User, error)
}

type PostService interface {
	GetByID(context.Context, int64) (*model.Post, error)
}

type Handler struct {
	UserHandler *UserHandler
	PostHandler *PostHandler
}

func NewHandler(user UserService, post PostService) Handler {
	return Handler{
		UserHandler: NewUserHandler(user),
		PostHandler: NewPostHandler(post),
	}
}

func readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1_048_578 // 1mb
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	return decoder.Decode(data)
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func jsonResponse(w http.ResponseWriter, status int, data any) error {
	type envelope struct {
		Data any `json:"data"`
	}
	return writeJSON(w, status, &envelope{Data: data})
}

func writeJSONError(w http.ResponseWriter, status int, message string) error {
	type envelope struct {
		Error string `json:"error"`
	}

	return writeJSON(w, status, &envelope{Error: message})
}

func internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	//app.logger.Errorw("internal error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusInternalServerError, "the server encountered a problem")
}

func forbiddenResponse(w http.ResponseWriter, r *http.Request) {
	//app.logger.Warnw("forbidden", "method", r.Method, "path", r.URL.Path, "error")

	writeJSONError(w, http.StatusForbidden, "forbidden")
}

func badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	//app.logger.Warnf("bad request", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusBadRequest, err.Error())
}

func conflictResponse(w http.ResponseWriter, r *http.Request, err error) {
	//app.logger.Errorf("conflict response", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusConflict, err.Error())
}

func notFoundResponse(w http.ResponseWriter, r *http.Request, err error) {
	//app.logger.Warnf("not found error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusNotFound, "not found")
}

func unauthorizedErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	//app.logger.Warnf("unauthorized error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	writeJSONError(w, http.StatusUnauthorized, "unauthorized")
}

func unauthorizedBasicErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	//app.logger.Warnf("unauthorized basic error", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)

	writeJSONError(w, http.StatusUnauthorized, "unauthorized")
}

func rateLimitExceededResponse(w http.ResponseWriter, r *http.Request, retryAfter string) {
	//app.logger.Warnw("rate limit exceeded", "method", r.Method, "path", r.URL.Path)

	w.Header().Set("Retry-After", retryAfter)

	writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded, retry after: "+retryAfter)
}
