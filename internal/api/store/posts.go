package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/model"
	"github.com/lib/pq"
)

type PostStore interface {
	GetByID(context.Context, int64) (*model.Post, error)
}
type postStore struct {
	db *sql.DB
}

// Explicitly ensuring that postStore adheres to the PostStore interface
var _ PostStore = (*postStore)(nil)

func (s *postStore) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	query := `
		SELECT id, user_id, title, content, created_at,  updated_at, tags, version
		FROM posts
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var post model.Post
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID,
		&post.UserID,
		&post.Title,
		&post.Content,
		&post.CreatedAt,
		&post.UpdatedAt,
		pq.Array(&post.Tags),
		&post.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, common.ErrNotFound
		default:
			return nil, err
		}
	}

	return &post, nil
}
