package store

import (
	"context"
	"database/sql"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/model"
)

type UserStore interface {
	GetByID(context.Context, int64) (*model.User, error)
}

type userStore struct {
	db *sql.DB
}

// Explicitly ensuring that userStore adheres to the UserStore interface
var _ UserStore = (*userStore)(nil)

func (s *userStore) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	query := `
		SELECT users.id, username, email, password, created_at, roles.*
		FROM users
		JOIN roles ON (users.role_id = roles.id)
		WHERE users.id = $1 AND is_active = true
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	user := &model.User{}
	err := s.db.QueryRowContext(
		ctx,
		query,
		userID,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password.Hash,
		&user.CreatedAt,
		&user.Role.ID,
		&user.Role.Name,
		&user.Role.Level,
		&user.Role.Description,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrNotFound
		default:
			return nil, err
		}
	}

	return user, nil
}
