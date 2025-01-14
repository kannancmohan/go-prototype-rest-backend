package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
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

func ApplyDBMigrations(db *sql.DB) error {
	driver, err := pgMigrate.WithInstance(db, &pgMigrate.Config{})
	if err != nil {
		return err
	}

	rootDir, _ := common.GetRootDir()
	migrationDir := "file://" + rootDir + "/cmd/migrate/migrations"
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

var (
	dbContainerInstances sync.Map // Map to store containers by schema name
	mutex                sync.Mutex
)

type ContainerInfo struct {
	Container testcontainers.Container
	DB        *sql.DB
}

type DBCleanupFunc func(ctx context.Context) error

func StartDBTestContainer(schemaName string) (*sql.DB, DBCleanupFunc, error) {

	mutex.Lock()
	defer mutex.Unlock()

	ctx := context.Background()

	// Check if a container for this schema already exists
	if instance, ok := dbContainerInstances.Load(schemaName); ok {
		info := instance.(*ContainerInfo)
		return info.DB, func(ctx context.Context) error { return nil }, nil // No-op cleanup for reused container
	}

	container, err := createTestContainer(ctx, schemaName, "test", "test")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start Postgres container: %w", err)
	}

	db, err := createDBInstance(ctx, container, schemaName, "test", "test")
	if err != nil {
		container.Terminate(ctx) // Ensure cleanup
		return nil, nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	dbContainerInstances.Store(schemaName, &ContainerInfo{
		Container: container,
		DB:        db,
	})

	cleanupFunc := func(ctx context.Context) error {
		mutex.Lock()
		defer mutex.Unlock()

		if instance, ok := dbContainerInstances.Load(schemaName); ok {
			dbContainerInstances.Delete(schemaName)

			info := instance.(*ContainerInfo)
			info.DB.Close() // Close the database connection

			err := info.Container.Terminate(ctx)
			if err != nil {
				return fmt.Errorf("failed to terminate Postgres container: %w", err)
			}
		}
		return nil
	}

	return db, cleanupFunc, nil
}

func createTestContainer(ctx context.Context, schemaName, dbUser, dbUserPwd string) (ctr *pgtc.PostgresContainer, err error) {

	ctr, err = pgtc.Run(
		ctx,
		"postgres:17-alpine",
		//pgtc.WithInitScripts(sqlScripts),
		pgtc.WithDatabase(schemaName),
		pgtc.WithUsername(dbUser),
		pgtc.WithPassword(dbUserPwd),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return ctr, err
	}
	return ctr, nil
}

func createDBInstance(ctx context.Context, container testcontainers.Container, schemaName, dbUser, dbUserPwd string) (*sql.DB, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, nat.Port("5432/tcp"))
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbUserPwd, host, port.Port(), schemaName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Postgres: %w", err)
	}

	return db, nil
}
