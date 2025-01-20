package service_test

import (
	"context"
	"testing"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/dto"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/service"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
	mockstore "github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store/mocks"
	"go.uber.org/mock/gomock"
)

func Test_CreatePost(t *testing.T) {

	commonSearchStore := func(tb testing.TB) *mockstore.MockPostSearchStore {
		tb.Helper()
		return mockstore.NewMockPostSearchStore(gomock.NewController(t))
	}

	testCases := []struct {
		name        string
		post        dto.CreatePostReq
		dbStore     func(tb testing.TB) *mockstore.MockPostDBStore
		msgStore    func(tb testing.TB) *mockstore.MockPostMessageBrokerStore
		searchStore func(tb testing.TB) *mockstore.MockPostSearchStore
		expErr      error
	}{
		{
			name: "error in dbstore",
			post: dto.CreatePostReq{
				Title:   "test1-title",
				Content: "test1-content",
				UserID:  1,
				Tags:    []string{"test"},
			},
			dbStore: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().Create(gomock.Any(), gomock.Any()).Return(common.NewErrorf(common.ErrorCodeUnknown, ""))
				return ms
			},
			msgStore: func(tb testing.TB) *mockstore.MockPostMessageBrokerStore {
				tb.Helper()
				return mockstore.NewMockPostMessageBrokerStore(gomock.NewController(t))
			},
			searchStore: commonSearchStore,
			expErr:      common.NewErrorf(common.ErrorCodeUnknown, ""),
		},
		{
			name: "error in messagebrokerstore",
			post: dto.CreatePostReq{
				Title:   "test1-title",
				Content: "test1-content",
				UserID:  1,
				Tags:    []string{"test"},
			},
			dbStore: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				return ms
			},
			msgStore: func(tb testing.TB) *mockstore.MockPostMessageBrokerStore {
				tb.Helper()
				ms := mockstore.NewMockPostMessageBrokerStore(gomock.NewController(t))
				ms.EXPECT().Created(gomock.Any(), gomock.Any()).Return(common.NewErrorf(common.ErrorCodeUnknown, ""))
				return ms
			},
			searchStore: commonSearchStore,
			expErr:      common.NewErrorf(common.ErrorCodeUnknown, ""),
		},
		{
			name: "success",
			post: dto.CreatePostReq{
				Title:   "test1-title",
				Content: "test1-content",
				UserID:  1,
				Tags:    []string{"test"},
			},
			dbStore: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				return ms
			},
			msgStore: func(tb testing.TB) *mockstore.MockPostMessageBrokerStore {
				tb.Helper()
				ms := mockstore.NewMockPostMessageBrokerStore(gomock.NewController(t))
				ms.EXPECT().Created(gomock.Any(), gomock.Any()).Return(nil)
				return ms
			},
			searchStore: commonSearchStore,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.NewPostService(tc.dbStore(t), tc.msgStore(t), tc.searchStore(t)).Create(ctx, tc.post)
			if tc.expErr == nil {
				if err != nil {
					t.Error("Post should be created without error, received error instead.", err.Error())
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

func Test_SearchPost(t *testing.T) {

	commonDBStore := func(tb testing.TB) *mockstore.MockPostDBStore {
		tb.Helper()
		return mockstore.NewMockPostDBStore(gomock.NewController(t))
	}

	commonMsgStore := func(tb testing.TB) *mockstore.MockPostMessageBrokerStore {
		tb.Helper()
		return mockstore.NewMockPostMessageBrokerStore(gomock.NewController(t))
	}

	testCases := []struct {
		name        string
		req         store.PostSearchReq
		dbStore     func(tb testing.TB) *mockstore.MockPostDBStore
		msgStore    func(tb testing.TB) *mockstore.MockPostMessageBrokerStore
		searchStore func(tb testing.TB) *mockstore.MockPostSearchStore
		expErr      error
	}{
		{
			name: "error in searchstore",
			req: store.PostSearchReq{
				Title: "test",
			},
			dbStore:  commonDBStore,
			msgStore: commonMsgStore,
			searchStore: func(tb testing.TB) *mockstore.MockPostSearchStore {
				tb.Helper()
				ms := mockstore.NewMockPostSearchStore(gomock.NewController(t))
				ms.EXPECT().Search(gomock.Any(), gomock.Any()).Return(store.PostSearchResp{}, common.NewErrorf(common.ErrorCodeUnknown, ""))
				return ms
			},
			expErr: common.NewErrorf(common.ErrorCodeUnknown, ""),
		},
		{
			name: "success",
			req: store.PostSearchReq{
				Title: "test",
			},
			dbStore:  commonDBStore,
			msgStore: commonMsgStore,
			searchStore: func(tb testing.TB) *mockstore.MockPostSearchStore {
				tb.Helper()
				ms := mockstore.NewMockPostSearchStore(gomock.NewController(t))
				ms.EXPECT().Search(gomock.Any(), gomock.Any()).Return(store.PostSearchResp{}, nil)
				return ms
			},
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.NewPostService(tc.dbStore(t), tc.msgStore(t), tc.searchStore(t)).Search(ctx, tc.req)
			if tc.expErr == nil {
				if err != nil {
					t.Error("Post search should not throw error, received error instead.", err.Error())
				}
			} else {
				receivedErr, receivedErrOk := err.(*common.Error)
				expectedErr, expectedErrOk := tc.expErr.(*common.Error)
				if !receivedErrOk || !expectedErrOk || receivedErr.Code() != expectedErr.Code() {
					t.Errorf("Post search should fail with error code %d, but received %d instead.", expectedErr.Code(), receivedErr.Code())
				}
			}
		})
	}
}
