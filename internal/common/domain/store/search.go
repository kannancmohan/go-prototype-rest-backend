package store

import (
	"context"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
)

// type PostSearchStore interface {
// 	Search(ctx context.Context, args internal.SearchParams) (internal.SearchResults, error)
// }

type PostSearchIndexStore interface {
	Delete(ctx context.Context, id string) error
	Index(ctx context.Context, task model.Post) error
}
