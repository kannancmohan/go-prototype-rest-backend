package handler_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/handler"
	mockservice "github.com/kannancmohan/go-prototype-rest-backend/internal/api/service/mocks"
	"go.uber.org/mock/gomock"
)

func Test_RegisterUserHandler(t *testing.T) {

	testCases := []struct {
		name           string
		body           io.Reader
		setup          func(tb testing.TB) *mockservice.MockUserService
		expectedStatus int
	}{
		{
			name: "invalid request json body",
			body: strings.NewReader(`{`),
			setup: func(tb testing.TB) *mockservice.MockUserService {
				tb.Helper()
				return mockservice.NewMockUserService(gomock.NewController(t))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid request body",
			body: strings.NewReader(`
			{ 
			"username": "test2",
			"email": "test2@test.com",
			"password": "test2@qq",
			}`),
			setup: func(tb testing.TB) *mockservice.MockUserService {
				tb.Helper()
				return mockservice.NewMockUserService(gomock.NewController(t))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service throws error",
			body: strings.NewReader(`
			{ 
			"username": "test3",
			"email": "test3@test.com",
			"password": "test3@qq",
			"role": "admin"
			}`),
			setup: func(tb testing.TB) *mockservice.MockUserService {
				tb.Helper()
				ms := mockservice.NewMockUserService(gomock.NewController(t))
				ms.EXPECT().CreateAndInvite(gomock.Any(), gomock.Any()).Return(nil, errors.New("service error"))
				return ms
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "success",
			body: strings.NewReader(`
			{ 
			"username": "test4",
			"email": "test4@test.com",
			"password": "test4@qq",
			"role": "admin"
			}`),
			setup: func(tb testing.TB) *mockservice.MockUserService {
				tb.Helper()
				ms := mockservice.NewMockUserService(gomock.NewController(t))
				ms.EXPECT().CreateAndInvite(gomock.Any(), gomock.Any()).Return(nil, nil)
				return ms
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", tc.body)

			handler.NewUserHandler(tc.setup(t)).RegisterUserHandler(w, r)

			if w.Code != tc.expectedStatus {
				t.Errorf("expected status code %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}
