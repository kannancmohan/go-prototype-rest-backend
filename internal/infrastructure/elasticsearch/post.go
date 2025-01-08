package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	esv8 "github.com/elastic/go-elasticsearch/v8"
	esv8api "github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
)

type postSearchIndexStore struct {
	client *esv8.Client
	index  string
}

func NewPostSearchIndexStore(client *esv8.Client, index string) *postSearchIndexStore {
	return &postSearchIndexStore{
		client: client,
		index:  index,
	}
}

func (p *postSearchIndexStore) Delete(ctx context.Context, id string) error {
	req := esv8api.DeleteRequest{
		Index:      p.index,
		DocumentID: id,
	}

	resp, err := req.Do(ctx, p.client)
	if err != nil {
		return fmt.Errorf("error deleting index %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("error deleting index post id=%s: %s", id, resp.String())
	}

	io.Copy(io.Discard, resp.Body) // TODO check if this is necessary ?

	return nil
}

func (p *postSearchIndexStore) Index(ctx context.Context, post model.Post) error {
	postId := strconv.FormatInt(post.ID, 10)
	body := store.IndexedPost{
		ID:        postId,
		Title:     post.Title,
		Content:   post.Content,
		UserID:    post.UserID,
		Tags:      post.Tags,
		CreatedAt: post.CreatedAt,
		Version:   post.Version,
	}

	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return fmt.Errorf("error marshalling post to JSON: %w", err)
	}

	req := esv8api.IndexRequest{
		Index:      p.index,
		Body:       &buf,
		DocumentID: postId,
		Refresh:    "true",
	}

	resp, err := req.Do(ctx, p.client)
	if err != nil {
		return fmt.Errorf("error indexing post %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("error indexing post id=%s: %s", postId, resp.String())
	}

	io.Copy(io.Discard, resp.Body) // TODO check if this is necessary ?

	return nil
}

func (p *postSearchIndexStore) Search(ctx context.Context, args store.PostSearchReq) (store.PostSearchResp, error) {
	if args.IsZero() {
		return store.PostSearchResp{}, common.NewErrorf(common.ErrorCodeBadRequest, "invalid PostSearchReq")
	}

	// set Default values for `sort` , `from` &  `size` if not provided
	if args.Sort == "" {
		args.Sort = "created_at" // Default sort by created_at
	}
	if args.Size <= 0 {
		args.Size = 10 // Default to 10 results
	}
	if args.From < 0 {
		args.From = 0 // Start from the first result
	}

	should := make([]interface{}, 0, 3)
	// "should" condition
	if args.Title != "" {
		should = append(should, map[string]interface{}{
			"match": map[string]interface{}{
				"title": args.Title,
			},
		})
	}

	if args.Content != "" {
		should = append(should, map[string]interface{}{
			"match": map[string]interface{}{
				"content": args.Content,
			},
		})
	}

	if len(args.Tags) > 0 {
		should = append(should, map[string]interface{}{
			"match": map[string]interface{}{
				"tags": args.Tags,
			},
		})
	}

	must := make([]interface{}, 0, 1)
	// "must" condition
	if args.UserID > 0 {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{
				"user_id": args.UserID,
			},
		})
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must":   must,   // Mandatory conditions
				"should": should, // Optional conditions
			},
		},
		"sort": []map[string]interface{}{
			{args.Sort: map[string]string{"order": "asc"}}, // Default to ascending order
		},
		"from": args.From,
		"size": args.Size,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return store.PostSearchResp{}, common.WrapErrorf(err, common.ErrorCodeUnknown, "error encoding query to JSON")
	}

	req := esv8api.SearchRequest{
		Index: []string{p.index},
		Body:  &buf,
	}

	resp, err := req.Do(ctx, p.client)
	if err != nil {
		return store.PostSearchResp{}, common.WrapErrorf(err, common.ErrorCodeUnknown, "error executing search request")
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return store.PostSearchResp{}, common.NewErrorf(common.ErrorCodeUnknown, "error in search response%s", resp.String())
	}

	var searchResult struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source store.IndexedPost `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return store.PostSearchResp{}, common.WrapErrorf(err, common.ErrorCodeUnknown, "error decoding search response")
	}

	results := make([]store.IndexedPost, len(searchResult.Hits.Hits))
	for i, hit := range searchResult.Hits.Hits {
		results[i] = hit.Source
	}

	return store.PostSearchResp{
		Results: results,
		Total:   searchResult.Hits.Total.Value,
	}, nil

}
