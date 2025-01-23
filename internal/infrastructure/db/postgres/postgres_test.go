//go:build !skip_docker_tests

package postgres_test

import (
	"context"
	"database/sql"
	"log"
	"os"
	"testing"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/db/postgres"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/testutils"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	var cleanupFunc testutils.DBCleanupFunc

	pgTest := testutils.NewTestPostgresContainer("testpg", "test", "test")
	container, cleanupFunc, err := pgTest.CreatePostgresTestContainer()
	if err != nil {
		log.Fatalf("Failed to start TestContainer: %v", err)
	}

	testDB, err = pgTest.CreatePostgresDBInstance(container)
	if err != nil {
		log.Fatalf("Failed to init postgres: %v", err)
	}

	if err := testutils.ApplyDBMigrations(testDB); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	code := m.Run()

	defer func() {
		if testDB != nil {
			testDB.Close()
		}
		if cleanupFunc != nil {
			if err := cleanupFunc(context.Background()); err != nil {
				log.Printf("Failed to clean up TestContainer: %v", err)
			}
		}
	}()

	os.Exit(code)
}

func TestRoleStore_GetByName(t *testing.T) {
	testCases := []struct {
		name     string
		roleName string
		expErr   error
	}{
		{
			name:     "success",
			roleName: "user",
		},
		{
			name:     "invalid role name",
			roleName: "unknown",
			expErr:   common.ErrNotFound,
		},
		{
			name:     "empty role name",
			roleName: "",
			expErr:   common.ErrNotFound,
		},
	}
	rStore := postgres.NewRoleDBStore(testDB, time.Second*10)
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			role, err := rStore.GetByName(ctx, tc.roleName)
			if tc.expErr == nil {
				if err != nil || role == nil || role.Name != tc.roleName {
					t.Errorf("expected role name '%s', but got '%s' (error: %v)", tc.roleName, role.Name, err)
				}
			} else {
				receivedErr, receivedErrOk := err.(*common.Error)
				expectedErr, expectedErrOk := tc.expErr.(*common.Error)
				if !receivedErrOk || !expectedErrOk || receivedErr.Code() != expectedErr.Code() {
					t.Errorf("Finding role should fail with error code %d, but received %d instead.", expectedErr.Code(), receivedErr.Code())
				}
			}
		})
	}
}

func TestUserStore_Create(t *testing.T) {
	testCases := []struct {
		name   string
		user   *model.User
		expErr error
	}{
		{
			name: "success",
			user: &model.User{
				Username: "test-user",
				Email:    "test-user@test.com",
				Password: model.Password{
					Hash: []byte("test_hashed_password"),
				},
			},
		},
		{
			name: "email already exists",
			user: &model.User{
				Username: "test-user1",
				Email:    "test-user@test.com",
				Password: model.Password{
					Hash: []byte("test_hashed_password"),
				},
			},
			expErr: common.NewErrorf(common.ErrorCodeBadRequest, ""),
		},
		{
			name: "username already exists",
			user: &model.User{
				Username: "test-user",
				Email:    "test-user1@test.com",
				Password: model.Password{
					Hash: []byte("test_hashed_password"),
				},
			},
			expErr: common.NewErrorf(common.ErrorCodeBadRequest, ""),
		},
		{
			name: "invalid role should throw unknown error code",
			user: &model.User{
				Username: "test-user2",
				Email:    "test-user2@test.com",
				Password: model.Password{
					Hash: []byte("test_hashed_password"),
				},
				Role: model.Role{
					Name: "test",
				},
			},
			expErr: common.NewErrorf(common.ErrorCodeUnknown, ""),
		},
	}
	uStore := postgres.NewUserDBStore(testDB, time.Second*10)
	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := uStore.Create(ctx, tc.user)
			if tc.expErr == nil {
				if err != nil {
					t.Error("User should be created without error, received error instead.", err.Error())
				}
			} else {
				receivedErr, receivedErrOk := err.(*common.Error)
				expectedErr, expectedErrOk := tc.expErr.(*common.Error)
				if !receivedErrOk || !expectedErrOk || receivedErr.Code() != expectedErr.Code() {
					t.Errorf("User creation should fail with error code %d, but received %d instead.", expectedErr.Code(), receivedErr.Code())
				}
			}
		})
	}
}

func TestPostStore_Create(t *testing.T) {

	ctx := context.Background()
	uStore := postgres.NewUserDBStore(testDB, time.Second*10)
	user := &model.User{Username: "user_post", Email: "user_post@test.com", Password: model.Password{
		Hash: []byte("test_hashed_password"),
	}}
	err := uStore.Create(ctx, user)
	if err != nil {
		t.Error("Perquisite failed: User creating failed in TestPostStore_Create")
	}

	testCases := []struct {
		name   string
		post   *model.Post
		expErr error
	}{
		{
			name: "success",
			post: &model.Post{
				Title:   "test post title1",
				Content: "test post content1",
				Tags:    []string{"test"},
				UserID:  user.ID,
			},
		},
		{
			name: "create post should fail on invalid user",
			post: &model.Post{
				Title:   "test post title1",
				Content: "test post content1",
				Tags:    []string{"test"},
				UserID:  0,
			},
			expErr: common.NewErrorf(common.ErrorCodeUnknown, ""),
		},
	}

	pStore := postgres.NewPostDBStore(testDB, time.Second*10)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := pStore.Create(ctx, tc.post)
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
