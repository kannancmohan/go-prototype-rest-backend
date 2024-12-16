package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
	"github.com/lib/pq"
)

type postStore struct {
	db     *sql.DB
	config *config.ApiConfig
}

func NewPostStore(db *sql.DB, cfg *config.ApiConfig) *postStore {
	return &postStore{db: db, config: cfg}
}

func (s *postStore) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	query := `
		SELECT id, user_id, title, content, created_at,  updated_at, tags, version
		FROM posts
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, s.config.SqlQueryTimeoutDuration)
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
