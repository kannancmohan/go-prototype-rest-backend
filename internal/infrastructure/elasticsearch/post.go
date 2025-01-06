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
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
)

type indexedPost struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	UserID    int64    `json:"user_id"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	Version   int      `json:"version"`
}

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
	body := indexedPost{
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
