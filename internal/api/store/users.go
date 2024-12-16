package store

import (
	"context"
	"database/sql"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
)

type userStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *userStore {
	return &userStore{db: db}
}

// Explicitly ensuring that userStore adheres to the UserStore interface
//var _ UserStore = (*userStore)(nil)

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
			return nil, common.ErrNotFound
		default:
			return nil, err
		}
	}

	return user, nil
}

func (s *userStore) Create(ctx context.Context, tx *sql.Tx, user *model.User) error {
	query := `
		INSERT INTO users (username, password, email, role_id) VALUES 
    ($1, $2, $3, (SELECT id FROM roles WHERE name = $4))
    RETURNING id, created_at
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	role := user.Role.Name
	if role == "" {
		role = "user"
	}

	err := tx.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Password.Hash,
		user.Email,
		role,
	).Scan(
		&user.ID,
		&user.CreatedAt,
	)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return common.ErrDuplicateEmail
		case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
			return common.ErrDuplicateUsername
		default:
			return err
		}
	}

	return nil
}
