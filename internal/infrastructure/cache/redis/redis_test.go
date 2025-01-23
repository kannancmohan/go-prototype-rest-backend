//go:build !skip_docker_tests

package redis_test

import (
	"context"
	"log"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	mockstore "github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store/mocks"
	redis_cache "github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/cache/redis"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/testutils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/mock/gomock"
)

var testRedisClient *redis.Client

func TestMain(m *testing.M) {
	var err error
	var cleanupFunc testutils.RedisCleanupFunc

	pgTest := testutils.NewTestRedisContainer()
	container, cleanupFunc, err := pgTest.CreateRedisTestContainer("")
	if err != nil {
		log.Fatalf("Failed to start TestContainer: %v", err)
	}

	connStr, err := pgTest.GetRedisConnectionString(container)
	if err != nil {
		log.Fatalf("Failed to get connection  string: %v", err)
	}

	testRedisClient, err = pgTest.CreateRedisInstance(connStr)
	if err != nil {
		log.Fatalf("Failed to init redis client: %v", err)
	}

	code := m.Run()

	defer func() {
		if testRedisClient != nil {
			testRedisClient.Close()
		}
		if cleanupFunc != nil {
			if err := cleanupFunc(context.Background()); err != nil {
				log.Printf("Failed to clean up TestContainer: %v", err)
			}
		}
	}()

	os.Exit(code)
}

func TestPostStore_Create(t *testing.T) {
	ctx := context.Background()
	testCases := []struct {
		name         string
		post         *model.Post
		dbStoreSetup func(tb testing.TB) *mockstore.MockPostDBStore
		expErr       error
	}{
		{
			name: "success",
			post: &model.Post{
				ID:      1,
				Title:   "test post title1",
				Content: "test post content1",
				Tags:    []string{"test"},
				UserID:  1,
			},
			dbStoreSetup: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				return ms
			},
		},
		{
			name: "caching should fail on dbstore error",
			post: &model.Post{
				ID:      2,
				Title:   "test post title2",
				Content: "test post content2",
				Tags:    []string{"test"},
				UserID:  1,
			},
			dbStoreSetup: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().Create(gomock.Any(), gomock.Any()).Return(common.NewErrorf(common.ErrorCodeUnknown, ""))
				return ms
			},
			expErr: common.NewErrorf(common.ErrorCodeUnknown, ""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := redis_cache.NewPostStore(testRedisClient, tc.dbStoreSetup(t), time.Minute*1).Create(ctx, tc.post)
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

func TestPostStore_Update(t *testing.T) {

	ctx := context.Background()

	// add test data to redis
	err := createTestPost(t, ctx, []model.Post{
		{
			ID:      2,
			Title:   "test-title-1",
			Content: "test-content-1",
			UserID:  200,
		},
	})

	if err != nil {
		t.Errorf("Adding test data for PostStore.update failed. Received error %v", err)
	}

	updatedTestPost := &model.Post{
		ID:      2,
		Title:   "test-title-updated-1",
		Content: "test-content-updated-1",
		UserID:  200,
	}

	testCases := []struct {
		name         string
		post         *model.Post
		dbStoreSetup func(tb testing.TB) *mockstore.MockPostDBStore
		expErr       error
		expResp      *model.Post
	}{
		{
			name: "success",
			post: updatedTestPost,
			dbStoreSetup: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().Update(gomock.Any(), gomock.Any()).Return(updatedTestPost, nil)
				return ms
			},
			expResp: updatedTestPost,
		},
		{
			name: "update caching should fail on dbstore error",
			post: &model.Post{
				ID:      1,
				Title:   "test-title-updated-1",
				Content: "test-content-updated-1",
				UserID:  200,
			},
			dbStoreSetup: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, common.NewErrorf(common.ErrorCodeUnknown, ""))
				return ms
			},
			expErr: common.NewErrorf(common.ErrorCodeUnknown, ""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			receivedResp, err := redis_cache.NewPostStore(testRedisClient, tc.dbStoreSetup(t), time.Minute*1).Update(ctx, tc.post)
			if tc.expErr == nil {
				if err != nil {
					t.Error("Post should be updated without error, received error instead.", err.Error())
				}
				if !reflect.DeepEqual(tc.expResp, receivedResp) {
					t.Errorf("Post update expected result:%v, but received %v instead.", tc.expResp, receivedResp)
				}
			} else {
				receivedErr, receivedErrOk := err.(*common.Error)
				expectedErr, expectedErrOk := tc.expErr.(*common.Error)
				if !receivedErrOk || !expectedErrOk || receivedErr.Code() != expectedErr.Code() {
					t.Errorf("Post update should fail with error code %d, but received %d instead.", expectedErr.Code(), receivedErr.Code())
				}
			}
		})
	}
}

func TestPostStore_Delete(t *testing.T) {

	ctx := context.Background()

	// add test data to redis
	err := createTestPost(t, ctx, []model.Post{
		{
			ID:      3,
			Title:   "test-title-1",
			Content: "test-content-1",
			UserID:  300,
		},
	})

	if err != nil {
		t.Errorf("Adding test data for PostStore.delete failed. Received error %v", err)
	}

	testCases := []struct {
		name         string
		postID       int64
		dbStoreSetup func(tb testing.TB) *mockstore.MockPostDBStore
		expErr       error
	}{
		{
			name:   "success",
			postID: 3,
			dbStoreSetup: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)
				return ms
			},
		},
		{
			name:   "delete caching should fail on dbstore error",
			postID: 3,
			dbStoreSetup: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(common.NewErrorf(common.ErrorCodeUnknown, ""))
				return ms
			},
			expErr: common.NewErrorf(common.ErrorCodeUnknown, ""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := redis_cache.NewPostStore(testRedisClient, tc.dbStoreSetup(t), time.Minute*1).Delete(ctx, tc.postID)
			if tc.expErr == nil {
				if err != nil {
					t.Error("Post should be deleted without error, received error instead.", err.Error())
				}
			} else {
				receivedErr, receivedErrOk := err.(*common.Error)
				expectedErr, expectedErrOk := tc.expErr.(*common.Error)
				if !receivedErrOk || !expectedErrOk || receivedErr.Code() != expectedErr.Code() {
					t.Errorf("Post delete should fail with error code %d, but received %d instead.", expectedErr.Code(), receivedErr.Code())
				}
			}
		})
	}
}

func TestPostStore_GetByID(t *testing.T) {

	ctx := context.Background()

	alreadyCachedPost := &model.Post{
		ID:      4,
		Title:   "test-title-1",
		Content: "test-content-1",
		UserID:  400,
	}

	newlyCachedPost := &model.Post{
		ID:      5,
		Title:   "test-title-1",
		Content: "test-content-1",
		UserID:  500,
	}

	// add test data to redis
	err := createTestPost(t, ctx, []model.Post{*alreadyCachedPost})

	if err != nil {
		t.Errorf("Adding test data for PostStore.GetByID failed. Received error %v", err)
	}

	testCases := []struct {
		name         string
		postID       int64
		dbStoreSetup func(tb testing.TB) *mockstore.MockPostDBStore
		expErr       error
		expResp      *model.Post
	}{
		{
			name:   "success - get already cached post",
			postID: alreadyCachedPost.ID,
			dbStoreSetup: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				return ms
			},
			expResp: alreadyCachedPost,
		},
		{
			name:   "success - get newly cached post",
			postID: newlyCachedPost.ID,
			dbStoreSetup: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(newlyCachedPost, nil)
				return ms
			},
			expResp: newlyCachedPost,
		},
		{
			name:   "GetByID should fail on dbstore error",
			postID: 6,
			dbStoreSetup: func(tb testing.TB) *mockstore.MockPostDBStore {
				tb.Helper()
				ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
				ms.EXPECT().GetByID(gomock.Any(), gomock.Any()).Return(nil, common.NewErrorf(common.ErrorCodeUnknown, ""))
				return ms
			},
			expErr: common.NewErrorf(common.ErrorCodeUnknown, ""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			receivedResp, err := redis_cache.NewPostStore(testRedisClient, tc.dbStoreSetup(t), time.Minute*1).GetByID(ctx, tc.postID)
			if tc.expErr == nil {
				if err != nil {
					t.Error("Post GetByID should be success, received error instead.", err.Error())
				}
				if !reflect.DeepEqual(tc.expResp, receivedResp) {
					t.Errorf("Post GetByID expected result:%v, but received %v instead.", tc.expResp, receivedResp)
				}
			} else {
				receivedErr, receivedErrOk := err.(*common.Error)
				expectedErr, expectedErrOk := tc.expErr.(*common.Error)
				if !receivedErrOk || !expectedErrOk || receivedErr.Code() != expectedErr.Code() {
					t.Errorf("Post GetByID should fail with error code %d, but received %d instead.", expectedErr.Code(), receivedErr.Code())
				}
			}
		})
	}
}

func createTestPost(t testing.TB, ctx context.Context, posts []model.Post) error {
	for _, p := range posts {
		ms := mockstore.NewMockPostDBStore(gomock.NewController(t))
		ms.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

		err := redis_cache.NewPostStore(testRedisClient, ms, time.Minute*1).Create(ctx, &p)
		if err != nil {
			return err
		}
	}
	return nil
}
