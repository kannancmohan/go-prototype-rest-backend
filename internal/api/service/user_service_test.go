package service_test

import (
	"context"
	"testing"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/dto"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/service"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
	mockstore "github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store/mocks"
	"go.uber.org/mock/gomock"
)

func Test_CreateAndInvite(t *testing.T) {

	testCases := []struct {
		name    string
		req     dto.CreateUserReq
		dbStore func(tb testing.TB) *mockstore.MockUserDBStore
		expErr  error
	}{
		{
			name: "invalid password in request",
			req: dto.CreateUserReq{
				Username: "test",
				Email:    "test@test",
				Role:     "admin",
			},
			dbStore: func(tb testing.TB) *mockstore.MockUserDBStore {
				tb.Helper()
				return mockstore.NewMockUserDBStore(gomock.NewController(t))
			},
			expErr: common.NewErrorf(common.ErrorCodeBadRequest, ""),
		},
		{
			name: "error in dbstore",
			req: dto.CreateUserReq{
				Username: "test",
				Password: "test@test",
				Email:    "test@test",
				Role:     "admin",
			},
			dbStore: func(tb testing.TB) *mockstore.MockUserDBStore {
				tb.Helper()
				ms := mockstore.NewMockUserDBStore(gomock.NewController(t))
				ms.EXPECT().Create(gomock.Any(), gomock.Any()).Return(common.NewErrorf(common.ErrorCodeUnknown, ""))
				return ms
			},
			expErr: common.NewErrorf(common.ErrorCodeUnknown, ""),
		},
		{
			name: "success",
			req: dto.CreateUserReq{
				Username: "test",
				Password: "test@test",
				Email:    "test@test",
				Role:     "admin",
			},
			dbStore: func(tb testing.TB) *mockstore.MockUserDBStore {
				tb.Helper()
				ms := mockstore.NewMockUserDBStore(gomock.NewController(t))
				ms.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				return ms
			},
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.NewUserService(tc.dbStore(t)).CreateAndInvite(ctx, tc.req)
			if tc.expErr == nil {
				if err != nil {
					t.Error("User should be created without error, received error instead.", err.Error())
				}
			} else {
				receivedErr, receivedErrOk := err.(*common.Error)
				expectedErr, expectedErrOk := tc.expErr.(*common.Error)
				if !receivedErrOk || !expectedErrOk || receivedErr.Code() != expectedErr.Code() {
					t.Errorf("Post creation should fail with error code %d, but received %d instead.", expectedErr.Code(), receivedErr.Code())
				}
			}
		})
	}
}
