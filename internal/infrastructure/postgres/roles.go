package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/config"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/domain/model"
)

type roleStore struct {
	db     *sql.DB
	config *config.ApiConfig
}

func NewRoleStore(db *sql.DB, cfg *config.ApiConfig) *roleStore {
	return &roleStore{db: db, config: cfg}
}

func (s *roleStore) GetByName(ctx context.Context, slug string) (*model.Role, error) {
	query := `SELECT id, name, description, level FROM roles WHERE name = $1`

	role := &model.Role{}
	err := s.db.QueryRowContext(ctx, query, slug).Scan(&role.ID, &role.Name, &role.Description, &role.Level)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, common.ErrNotFound
		default:
			return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "role not found")
		}
	}

	return role, nil
}
