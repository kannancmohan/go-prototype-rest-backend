package store

import (
	"database/sql"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/adapter"
)

var (
	QueryTimeoutDuration = time.Second * 5
)

func NewStorage(db *sql.DB) adapter.Storage {
	return adapter.Storage{
		Posts: NewPostStore(db),
		Users: &userStore{db},
		Roles: &roleStore{db},
	}
}
