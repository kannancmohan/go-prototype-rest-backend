package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

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

func NewPostSearchIndexStore(ctx context.Context, client *esv8.Client, index string, autoCreateIndex bool) (*postSearchIndexStore, error) {

	err := checkElasticIndexExists(ctx, client, index)
	if err != nil {
		if autoCreateIndex {
			if err := createIndexWithMapping(ctx, client, index); err != nil && !isResourceAlreadyExistsError(err) {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &postSearchIndexStore{
		client: client,
		index:  index,
	}, nil
}

func (p *postSearchIndexStore) Delete(ctx context.Context, id string) error {
	req := esv8api.DeleteRequest{
		Index:      p.index,
		DocumentID: id,
	}

	resp, err := req.Do(ctx, p.client)
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "error deleting index")
	}
	defer resp.Body.Close()

	if resp.IsError() {
		if resp.StatusCode == http.StatusNotFound {
			return common.ErrNotFound
		}
		return common.NewErrorf(common.ErrorCodeUnknown, "error deleting index post id=%s: err=%s", id, resp.String())
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
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "error marshalling post to JSON")
	}

	req := esv8api.IndexRequest{
		Index:      p.index,
		Body:       &buf,
		DocumentID: postId,
		Refresh:    "true",
	}

	resp, err := req.Do(ctx, p.client)
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "error indexing post")
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return common.NewErrorf(common.ErrorCodeUnknown, "error indexing post id=%s: err=%s", postId, resp.String())
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
			"terms": map[string]interface{}{
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

func checkElasticIndexExists(ctx context.Context, client *esv8.Client, index string) error {
	resp, err := esv8api.IndicesExistsRequest{
		Index: []string{index},
	}.Do(ctx, client)

	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "error checking if index exists")
	}
	defer resp.Body.Close()

	if resp.IsError() {
		if resp.StatusCode == http.StatusNotFound {
			return common.NewErrorf(common.ErrorCodeNotFound, "index %s does not exist", index)
		}
		return common.NewErrorf(common.ErrorCodeUnknown, "error checking if index exists: %s", resp.String())
	}
	return nil
}

func isResourceAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "resource_already_exists_exception")
}

func createIndexWithMapping(ctx context.Context, client *esv8.Client, index string) error {
	esFieldMappings := `
	{
		"mappings": {
			"properties": {
				"created_at": {"type": "date", "ignore_malformed": true}
			}
		}
	}`

	resp, err := esv8api.IndicesCreateRequest{
		Index: index,
		Body:  strings.NewReader(esFieldMappings),
	}.Do(ctx, client)

	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "error creating index or setting mapping")
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return common.NewErrorf(common.ErrorCodeUnknown, "error creating index: %s", resp.String())
	}

	return nil
}
