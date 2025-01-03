package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	api_common "github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/lib/pq"
)

type postStore struct {
	db                      *sql.DB
	sqlQueryTimeoutDuration time.Duration
}

func NewPostStore(db *sql.DB, sqlQueryTimeoutDuration time.Duration) *postStore {
	return &postStore{db: db, sqlQueryTimeoutDuration: sqlQueryTimeoutDuration}
}

func (s *postStore) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	query := `
		SELECT id, user_id, title, content, created_at,  updated_at, tags, version
		FROM posts
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, s.sqlQueryTimeoutDuration)
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
			return nil, api_common.ErrNotFound
		default:
			return nil, api_common.WrapErrorf(err, api_common.ErrorCodeUnknown, "post not found")
		}
	}

	return &post, nil
}

func (s *postStore) Create(ctx context.Context, post *model.Post) error {
	query := `
	INSERT INTO posts (content, title, user_id, tags)
	VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at
`

	ctx, cancel := context.WithTimeout(ctx, s.sqlQueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(
		ctx,
		query,
		post.Content,
		post.Title,
		post.UserID,
		pq.Array(post.Tags),
	).Scan(
		&post.ID,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err != nil {
		return api_common.WrapErrorf(err, api_common.ErrorCodeUnknown, "create post")
	}

	return nil
}

func (s *postStore) Update(ctx context.Context, post *model.Post) (*model.Post, error) {
	var updatedPost model.Post

	query := `UPDATE posts SET updated_at = NOW(), version = version + 1`
	args := []interface{}{}
	argIndex := 1

	// Dynamically add fields to update
	if post.Title != "" {
		query += `, title = $` + fmt.Sprint(argIndex)
		args = append(args, post.Title)
		argIndex++
	}

	if post.Content != "" {
		query += `, content = $` + fmt.Sprint(argIndex)
		args = append(args, post.Content)
		argIndex++
	}

	if len(post.Tags) > 0 {
		query += `, tags = $` + fmt.Sprint(argIndex)
		args = append(args, pq.Array(post.Tags))
		argIndex++
	}

	query += ` WHERE id = $` + fmt.Sprint(argIndex)
	args = append(args, post.ID)

	query += ` RETURNING id, user_id, title, content, created_at, updated_at, tags, version`

	ctx, cancel := context.WithTimeout(ctx, s.sqlQueryTimeoutDuration)
	defer cancel()

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&updatedPost.ID,
		&updatedPost.UserID,
		&updatedPost.Title,
		&updatedPost.Content,
		&updatedPost.CreatedAt,
		&updatedPost.UpdatedAt,
		pq.Array(&updatedPost.Tags),
		&updatedPost.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, api_common.ErrNotFound
		default:
			return nil, api_common.WrapErrorf(err, api_common.ErrorCodeUnknown, "update post")
		}
	}

	return &updatedPost, nil
}
