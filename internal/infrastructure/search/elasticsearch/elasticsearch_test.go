//go:build !skip_docker_tests

package elasticsearch_test

import (
	"context"
	"log"
	"os"
	"slices"
	"testing"

	esv8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/infrastructure/search/elasticsearch"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/testutils"
)

var testESClient *esv8.Client

func TestMain(m *testing.M) {
	var err error
	var cleanupFunc testutils.ESCleanupFunc

	pgTest := testutils.NewTestElasticsearchContainer()
	container, cleanupFunc, err := pgTest.CreateElasticsearchTestContainer("")
	if err != nil {
		log.Fatalf("Failed to start elasticsearch TestContainer: %v", err)
	}
	testESClient, err = pgTest.CreateElasticsearchInstance(container)
	if err != nil {
		log.Fatalf("Failed to init elasticsearch: %v", err)
	}

	code := m.Run()

	if cleanupFunc != nil {
		if err := cleanupFunc(context.Background()); err != nil {
			log.Printf("Failed to clean up Elasticsearch TestContainer: %v", err)
		}
	}

	os.Exit(code)

}

func TestPostSearchIndexStore_Index(t *testing.T) {
	testCases := []struct {
		name   string
		post   model.Post
		expErr error
	}{
		{
			name: "success",
			post: model.Post{
				ID:      1,
				Title:   "test-title",
				Content: "test-content",
				UserID:  1,
			},
		},
	}

	store, _ := elasticsearch.NewPostSearchIndexStore(testESClient, "test_posts_index")
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := store.Index(ctx, tc.post)
			if tc.expErr == nil {
				if err != nil {
					t.Errorf("Indexing post should be success. Received error %v instead.", err)
				}
			} else {
				receivedErr, receivedErrOk := err.(*common.Error)
				expectedErr, expectedErrOk := tc.expErr.(*common.Error)
				if !receivedErrOk || !expectedErrOk || receivedErr.Code() != expectedErr.Code() {
					t.Errorf("Indexing post should fail with error code %d, but received %d instead.", expectedErr.Code(), receivedErr.Code())
				}
			}
		})
	}

}

func TestPostSearchIndexStore_Delete(t *testing.T) {

	testCases := []struct {
		name   string
		postID string
		expErr error
	}{
		{
			name:   "unknown id",
			postID: "unknown_id",
			expErr: common.NewErrorf(common.ErrorCodeNotFound, ""),
		},
		{
			name:   "success",
			postID: "123",
		},
	}

	store, _ := elasticsearch.NewPostSearchIndexStore(testESClient, "test_posts_index")
	ctx := context.Background()

	// add test data to elasticsearch
	err := store.Index(ctx, model.Post{
		ID:      123,
		Content: "test",
	})
	if err != nil {
		t.Errorf("Adding test data for delete index failed. Received error %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := store.Delete(ctx, tc.postID)
			if tc.expErr == nil {
				if err != nil {
					t.Errorf("Delete post index should be success. Received error %v instead.", err)
				}
			} else {
				receivedErr, receivedErrOk := err.(*common.Error)
				expectedErr, expectedErrOk := tc.expErr.(*common.Error)
				if !receivedErrOk || !expectedErrOk || receivedErr.Code() != expectedErr.Code() {
					t.Errorf("Deleting index post should fail with error code %d, but received %d instead.", expectedErr.Code(), receivedErr.Code())
				}
			}
		})
	}

}

func TestPostSearchIndexStore_Search(t *testing.T) {

	testCases := []struct {
		name       string
		req        store.PostSearchReq
		expErr     error
		expPostIds []string
	}{
		{
			name:   "invalid search request",
			req:    store.PostSearchReq{},
			expErr: common.NewErrorf(common.ErrorCodeBadRequest, ""),
		},
		{
			name: "search with tag",
			req: store.PostSearchReq{
				Tags: []string{"tag_common"},
			},
			expPostIds: []string{"1234", "4321"},
		},
		{
			name: "search with user",
			req: store.PostSearchReq{
				UserID: 100,
			},
			expPostIds: []string{"1234", "4321"},
		},
	}

	store, _ := elasticsearch.NewPostSearchIndexStore(testESClient, "test_posts_index")
	ctx := context.Background()

	// add two test data to elasticsearch
	err := store.Index(ctx, model.Post{
		ID:      1234,
		Content: "test-content-1234",
		Title:   "test-title-1234",
		Tags:    []string{"tag_common", "tag1"},
		UserID:  100,
	})
	err = store.Index(ctx, model.Post{
		ID:      4321,
		Content: "test-content-4321",
		Title:   "test-title-4321",
		Tags:    []string{"tag_common", "tag2"},
		UserID:  100,
	})

	if err != nil {
		t.Errorf("Adding test data for search index failed. Received error %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := store.Search(ctx, tc.req)
			if tc.expErr == nil {
				if err != nil {
					t.Errorf("Search post index should be success. Received error %v instead.", err)
				}
				receivedPostIds := make([]string, 0, len(resp.Results))
				for _, post := range resp.Results {
					receivedPostIds = append(receivedPostIds, post.ID)
				}
				if slices.Compare(tc.expPostIds, receivedPostIds) != 0 {
					t.Errorf("Search post index should be success. Expected postIds:%v , But Received: %v instead.", tc.expPostIds, receivedPostIds)
				}
			} else {
				receivedErr, receivedErrOk := err.(*common.Error)
				expectedErr, expectedErrOk := tc.expErr.(*common.Error)
				if !receivedErrOk || !expectedErrOk || receivedErr.Code() != expectedErr.Code() {
					t.Errorf("Search index should fail with error code %d, but received %d instead.", expectedErr.Code(), receivedErr.Code())
				}
			}
		})
	}

}
