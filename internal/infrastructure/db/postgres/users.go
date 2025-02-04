package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
)

type userStore struct {
	db                      *sql.DB
	sqlQueryTimeoutDuration time.Duration
}

func NewUserDBStore(db *sql.DB, sqlQueryTimeoutDuration time.Duration) *userStore {
	return &userStore{db: db, sqlQueryTimeoutDuration: sqlQueryTimeoutDuration}
}

func (s *userStore) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	query := `
		SELECT users.id, username, email, password, created_at, roles.*
		FROM users
		JOIN roles ON (users.role_id = roles.id)
		WHERE users.id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, s.sqlQueryTimeoutDuration)
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
			return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "user not found")
		}
	}

	return user, nil
}

func (s *userStore) GetByEmail(ctx context.Context, userEmail string) (*model.User, error) {
	query := `
		SELECT users.id, username, users.email, password, created_at, roles.*
		FROM users
		JOIN roles ON (users.role_id = roles.id)
		WHERE users.email = $1
	`

	ctx, cancel := context.WithTimeout(ctx, s.sqlQueryTimeoutDuration)
	defer cancel()

	user := &model.User{}
	err := s.db.QueryRowContext(
		ctx,
		query,
		userEmail,
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
			return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "user not found")
		}
	}

	return user, nil
}

func (s *userStore) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (username, password, email, role_id) VALUES 
    ($1, $2, $3, (SELECT id FROM roles WHERE name = $4))
    RETURNING id, created_at
	`

	ctx, cancel := context.WithTimeout(ctx, s.sqlQueryTimeoutDuration)
	defer cancel()

	role := user.Role.Name
	if role == "" {
		role = "user"
	}

	err := s.db.QueryRowContext(
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
			return common.NewErrorf(common.ErrorCodeBadRequest, "email already exists")
		case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
			return common.NewErrorf(common.ErrorCodeBadRequest, "username already exists")
		default:
			return common.WrapErrorf(err, common.ErrorCodeUnknown, "create user")
		}
	}

	return nil
}

func (s *userStore) Update(ctx context.Context, user *model.User) (*model.User, error) {
	var updatedUser *model.User

	err := withTx(s.db, ctx, func(tx *sql.Tx) error {
		query := `
			UPDATE users
			SET updated_at = NOW()
		`
		args := []interface{}{}
		argIndex := 1

		// Dynamically add fields to update
		if user.Username != "" {
			query += `, username = $` + fmt.Sprint(argIndex)
			args = append(args, user.Username)
			argIndex++
		}

		if len(user.Password.Hash) > 0 {
			query += `, password = $` + fmt.Sprint(argIndex)
			args = append(args, user.Password.Hash)
			argIndex++
		}

		if user.Email != "" {
			query += `, email = $` + fmt.Sprint(argIndex)
			args = append(args, user.Email)
			argIndex++
		}

		if user.Role.Name != "" {
			query += `, role_id = (SELECT id FROM roles WHERE name = $` + fmt.Sprint(argIndex) + `)`
			args = append(args, user.Role.Name)
			argIndex++
		}

		query += ` WHERE id = $` + fmt.Sprint(argIndex)
		args = append(args, user.ID)

		query += ` RETURNING id, username, email, role_id, created_at, updated_at`

		var roleID int
		err := tx.QueryRowContext(ctx, query, args...).Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&roleID,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			switch {
			case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
				return common.NewErrorf(common.ErrorCodeBadRequest, "email already exists")
			case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
				return common.NewErrorf(common.ErrorCodeBadRequest, "username already exists")
			case errors.Is(err, sql.ErrNoRows):
				return common.ErrNotFound
			default:
				return common.WrapErrorf(err, common.ErrorCodeUnknown, "update user")
			}
		}

		var roleName string
		err = tx.QueryRowContext(ctx, `SELECT name FROM roles WHERE id = $1`, roleID).Scan(&roleName)
		if err != nil {
			return common.WrapErrorf(err, common.ErrorCodeUnknown, "fetch role name for user")
		}

		user.Role = model.Role{Name: roleName}
		updatedUser = user
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedUser, nil
}

func (s *userStore) Delete(ctx context.Context, userID int64) error {
	query := `DELETE FROM users WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, s.sqlQueryTimeoutDuration)
	defer cancel()

	res, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "delete user")
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return common.WrapErrorf(err, common.ErrorCodeUnknown, "delete user: rows affected error")
	}

	if rows == 0 {
		return common.ErrNotFound
	}

	return nil
}
