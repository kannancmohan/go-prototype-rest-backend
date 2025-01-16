package handler_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler"
	mockservice "github.com/kannancmohan/go-prototype-rest-backend/internal/api/service/mocks"
	"go.uber.org/mock/gomock"
)

func Test_CreatePostHandler(t *testing.T) {

	testCases := []struct {
		name           string
		body           io.Reader
		setup          func(tb testing.TB) *mockservice.MockPostService
		expectedStatus int
	}{
		{
			name: "invalid request json body",
			body: strings.NewReader(`{`),
			setup: func(tb testing.TB) *mockservice.MockPostService {
				tb.Helper()
				return mockservice.NewMockPostService(gomock.NewController(t))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid request body",
			body: strings.NewReader(`
			{ 
			"user_id" : 1, 
			"content": "test-post-content",
			"tags": ["test"]
			}`),
			setup: func(tb testing.TB) *mockservice.MockPostService {
				tb.Helper()
				return mockservice.NewMockPostService(gomock.NewController(t))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service throws error",
			body: strings.NewReader(`
			{ 
			"user_id" : 1, 
			"title": "test-post-title", 
			"content": "test-post-content",
			"tags": ["test"]
			}`),
			setup: func(tb testing.TB) *mockservice.MockPostService {
				tb.Helper()
				ms := mockservice.NewMockPostService(gomock.NewController(t))
				ms.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, errors.New("service error"))
				return ms
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "success",
			body: strings.NewReader(`
			{ 
			"user_id" : 1, 
			"title": "test-post-title", 
			"content": "test-post-content",
			"tags": ["test"]
			}`),
			setup: func(tb testing.TB) *mockservice.MockPostService {
				tb.Helper()
				ms := mockservice.NewMockPostService(gomock.NewController(t))
				ms.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, nil)
				return ms
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", tc.body)

			handler.NewPostHandler(tc.setup(t)).CreatePostHandler(w, r)

			if w.Code != tc.expectedStatus {
				t.Errorf("expected status code %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}

func Test_UpdatePostHandler(t *testing.T) {

	testCases := []struct {
		name           string
		postId         string
		body           io.Reader
		setup          func(tb testing.TB) *mockservice.MockPostService
		expectedStatus int
	}{
		{
			name:   "invalid postId",
			postId: "invalid",
			body: strings.NewReader(`
			{ 
			"title": "test-post-title", 
			"content": "test-post-content",
			"tags": ["test"]
			}`),
			setup: func(tb testing.TB) *mockservice.MockPostService {
				tb.Helper()
				return mockservice.NewMockPostService(gomock.NewController(t))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid request json body",
			postId: "1",
			body:   strings.NewReader(`{`),
			setup: func(tb testing.TB) *mockservice.MockPostService {
				tb.Helper()
				return mockservice.NewMockPostService(gomock.NewController(t))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid request body",
			postId: "2",
			body: strings.NewReader(`
			{ 
			}`),
			setup: func(tb testing.TB) *mockservice.MockPostService {
				tb.Helper()
				return mockservice.NewMockPostService(gomock.NewController(t))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "service throws error",
			postId: "3",
			body: strings.NewReader(`
			{ 
			"title": "test-post-title", 
			"content": "test-post-content",
			"tags": ["test"]
			}`),
			setup: func(tb testing.TB) *mockservice.MockPostService {
				tb.Helper()
				ms := mockservice.NewMockPostService(gomock.NewController(t))
				ms.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, errors.New("service error"))
				return ms
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "success",
			postId: "4",
			body: strings.NewReader(`
			{ 
			"title": "test-post-title", 
			"content": "test-post-content",
			"tags": ["test"]
			}`),
			setup: func(tb testing.TB) *mockservice.MockPostService {
				tb.Helper()
				ms := mockservice.NewMockPostService(gomock.NewController(t))
				ms.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, nil)
				return ms
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/posts/"+tc.postId, tc.body)

			router := chi.NewRouter() //TODO instead of using a router, check an alternate to pass postID in url
			router.HandleFunc("/posts/{postID}", handler.NewPostHandler(tc.setup(t)).UpdatePostHandler)
			router.ServeHTTP(w, r)

			if w.Code != tc.expectedStatus {
				t.Errorf("expected status code %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
