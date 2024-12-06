package store

import (
	"context"
	"database/sql"
)

type Role struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Level       int    `json:"level"`
}

type RoleStore interface {
	GetByName(context.Context, string) (*Role, error)
}

type roleStore struct {
	db *sql.DB
}

// Explicitly ensuring that roleStore adheres to the RoleStore interface
var _ RoleStore = (*roleStore)(nil)

func (s *roleStore) GetByName(ctx context.Context, slug string) (*Role, error) {
	query := `SELECT id, name, description, level FROM roles WHERE name = $1`

	role := &Role{}
	err := s.db.QueryRowContext(ctx, query, slug).Scan(&role.ID, &role.Name, &role.Description, &role.Level)
	if err != nil {
		return nil, err
	}

	return role, nil
}
