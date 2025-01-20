package store

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
)

//go:generate mockgen -destination=mocks/mock_search.go -package=mockstore github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store PostSearchStore,PostSearchIndexStore

type IndexedPost struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	UserID    int64    `json:"user_id"`
	Tags      []string `json:"tags"`
	CreatedAt string   `json:"created_at"`
	Version   int      `json:"version"`
}

type PostSearchReq struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	UserID  int64    `json:"user_id"`
	Tags    []string `json:"tags"`
	Sort    string   `json:"sort"` // Field to sort by, e.g., "created_at"
	From    int      `json:"from"` // Starting index for pagination
	Size    int      `json:"size"` // Number of results to return

}

func (p PostSearchReq) IsZero() bool {
	return p.Title == "" &&
		p.Content == "" && p.UserID < 1 && len(p.Tags) < 1
}

type PostSearchResp struct {
	Results []IndexedPost `json:"results"`
	Total   int           `json:"total"`
}

type PostSearchStore interface {
	Search(ctx context.Context, args PostSearchReq) (PostSearchResp, error)
}

type PostSearchIndexStore interface {
	Delete(ctx context.Context, id string) error
	Index(ctx context.Context, task model.Post) error
}
