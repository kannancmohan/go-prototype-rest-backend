package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/golang-migrate/migrate/v4"
	pgMigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common"
	"github.com/testcontainers/testcontainers-go"
	pgtc "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func getMigrationSourcePath() string {
	path, exists := os.LookupEnv("MIGRATIONS_PATH")
	if exists {
		if strings.HasPrefix(path, "file://") {
			return path
		}
		return "file://" + path
	}
	rootDir, _ := common.GetRootDir()
	return "file://" + rootDir + "/cmd/migrate/migrations"
}

func ApplyDBMigrationsWithDBUrl(dbUrl string) error {
	migrationDir := getMigrationSourcePath()
	m, err := migrate.New(migrationDir, dbUrl)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func ApplyDBMigrations(db *sql.DB) error {
	driver, err := pgMigrate.WithInstance(db, &pgMigrate.Config{})
	if err != nil {
		return err
	}
	migrationDir := getMigrationSourcePath()
	m, err := migrate.NewWithDatabaseInstance(
		migrationDir,
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

type testPostgresContainer struct {
	schemaName string
	dbUserName string
	dbPassword string
}

func NewTestPostgresContainer(schemaName, dbUserName, dbPassword string) *testPostgresContainer {
	return &testPostgresContainer{schemaName: schemaName, dbUserName: dbUserName, dbPassword: dbPassword}
}

type DBCleanupFunc func(ctx context.Context) error

func (p *testPostgresContainer) CreatePostgresTestContainer() (*pgtc.PostgresContainer, DBCleanupFunc, error) {

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ctr, err := pgtc.Run(
		timeoutCtx,
		"postgres:17-alpine",
		//pgtc.WithInitScripts(sqlScripts),
		pgtc.WithDatabase(p.schemaName),
		pgtc.WithUsername(p.dbUserName),
		pgtc.WithPassword(p.dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return ctr, func(ctx context.Context) error { return nil }, err
	}

	cleanupFunc := func(ctx context.Context) error {
		err := ctr.Terminate(ctx)
		if err != nil {
			return err
		}
		return nil
	}
	return ctr, cleanupFunc, nil
}

func (p *testPostgresContainer) CreatePostgresDBInstance(container testcontainers.Container) (*sql.DB, error) {
	ctx := context.Background()
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	host, err := container.Host(timeoutCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(timeoutCtx, nat.Port("5432/tcp"))
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", p.dbUserName, p.dbPassword, host, port.Port(), p.schemaName)

	dbTimeoutCtx, dbCancel := context.WithTimeout(ctx, 15*time.Second)
	defer dbCancel()

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Postgres: %w", err)
	}

	if err := db.PingContext(dbTimeoutCtx); err != nil {
		return nil, fmt.Errorf("failed to ping Postgres: %w", err)
	}

	return db, nil
}
