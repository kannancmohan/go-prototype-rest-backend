package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/dto"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())

	// Custom struct validation for checking updatePostPayload struct
	validate.RegisterStructValidation(func(sl validator.StructLevel) {
		payload := sl.Current().Interface().(updatePostPayload)
		if payload.Title == "" && payload.Content == "" && len(payload.Tags) < 1 {
			sl.ReportError(payload, "updatePostPayload", "updatePostPayload", "atleastone=title content tags", "")
		}
	}, updatePostPayload{})
}

type UserService interface {
	GetByID(context.Context, int64) (*model.User, error)
	CreateAndInvite(context.Context, dto.CreateUserReq) (*model.User, error)
	Update(context.Context, dto.UpdateUserReq) (*model.User, error)
	Delete(context.Context, int64) error
}

type PostService interface {
	GetByID(context.Context, int64) (*model.Post, error)
	Create(context.Context, dto.CreatePostReq) (*model.Post, error)
	Update(context.Context, dto.UpdatePostReq) (*model.Post, error)
	Delete(context.Context, int64) error
	Search(context.Context, store.PostSearchReq) (store.PostSearchResp, error)
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

func readJSONValid[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	payload, err := readJSON[T](w, r)
	if err != nil {
		return payload, err
	}
	if err := validate.Struct(payload); err != nil {
		return payload, common.WrapErrorf(err, common.ErrorCodeBadRequest, "json validation")
	}
	return payload, nil
}

func readJSON[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var v T
	maxBytes := 1_048_578 // 1mb
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&v); err != nil {
		return v, common.WrapErrorf(err, common.ErrorCodeBadRequest, "json decoder")
	}
	return v, nil
}

func renderResponse[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "json encoder")
	}
	return nil
}

type errorResponse struct {
	Error       string   `json:"error"`
	Validations []string `json:"validations,omitempty"`
}

func renderErrorResponse(w http.ResponseWriter, msg string, err error) {
	resp := errorResponse{Error: msg}
	status := http.StatusInternalServerError //default status
	logLevel := slog.LevelInfo               //default log level

	var ierr *common.Error
	if !errors.As(err, &ierr) {
		logLevel = slog.LevelError
		resp.Error = "internal error"
	} else {
		switch ierr.Code() {
		case common.ErrorCodeNotFound:
			status = http.StatusNotFound
		case common.ErrorCodeBadRequest:
			status = http.StatusBadRequest
			origErr := ierr.Unwrap()
			setValidationErrors(origErr, &resp)
		case common.ErrorCodeConflict:
			status = http.StatusConflict
		case common.ErrorCodeUnknown:
			fallthrough
		default:
			logLevel = slog.LevelError
			status = http.StatusInternalServerError
		}
	}
	logWithLevel(logLevel, err)
	renderResponse(w, status, resp)
}

func logWithLevel(level slog.Level, err error) {
	switch level {
	case slog.LevelDebug:
		slog.Debug(err.Error())
	case slog.LevelInfo:
		slog.Info(err.Error())
	case slog.LevelError:
		slog.Error(err.Error())
	default:
		slog.Info(err.Error()) // Default to Info level
	}
}

func setValidationErrors(origErr error, res *errorResponse) {
	if validationErrors, ok := origErr.(validator.ValidationErrors); ok {
		var messages []string
		for _, fieldError := range validationErrors {
			message := fmt.Sprintf("The field '%s' failed on the '%s' validation", fieldError.Field(), fieldError.Tag())
			messages = append(messages, message)
		}
		res.Validations = messages
	}
}

func getIntParam(param string, r *http.Request) (int64, error) {
	userID, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64) //TODO
	if err != nil {
		return userID, common.WrapErrorf(err, common.ErrorCodeBadRequest, "invalid request")
	}
	return userID, nil
}
