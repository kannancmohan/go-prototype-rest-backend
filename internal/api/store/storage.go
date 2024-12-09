package store

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("resource already exists")
	QueryTimeoutDuration = time.Second * 5
)

type Storage struct {
	Posts PostStore
	Users UserStore
	// Comments interface {
	// 	Create(context.Context, *Comment) error
	// 	GetByPostID(context.Context, int64) ([]Comment, error)
	// }
	// Followers interface {
	// 	Follow(ctx context.Context, userID, followerID int64) error
	// 	Unfollow(ctx context.Context, followerID, userID int64) error
	// }
	Roles RoleStore
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Posts: &postStore{db},
		Users: &userStore{db},
		// Comments:  &CommentStore{db},
		// Followers: &FollowerStore{db},
		Roles: &roleStore{db},
	}
}
