package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
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

var (
	dbContainerInstances sync.Map // Map to store containers by schema name
	dbMutex              sync.Mutex
)

type DBContainerInfo struct {
	Container testcontainers.Container
	DB        *sql.DB
}

type DBCleanupFunc func(ctx context.Context) error

func StartPostgresDBTestContainer(schemaName string) (*sql.DB, DBCleanupFunc, error) {

	dbMutex.Lock()
	defer dbMutex.Unlock()

	ctx := context.Background()

	// Check if a container for this schema already exists
	if instance, ok := dbContainerInstances.Load(schemaName); ok {
		info := instance.(*DBContainerInfo)
		return info.DB, func(ctx context.Context) error { return nil }, nil // No-op cleanup for reused container
	}

	container, err := createPostgresTestContainer(ctx, schemaName, "test", "test")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start Postgres container: %w", err)
	}

	db, err := createPostgresDBInstance(ctx, container, schemaName, "test", "test")
	if err != nil {
		container.Terminate(ctx) // Ensure cleanup
		return nil, nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	dbContainerInstances.Store(schemaName, &DBContainerInfo{
		Container: container,
		DB:        db,
	})

	cleanupFunc := func(ctx context.Context) error {
		dbMutex.Lock()
		defer dbMutex.Unlock()

		if instance, ok := dbContainerInstances.Load(schemaName); ok {
			dbContainerInstances.Delete(schemaName)

			info := instance.(*DBContainerInfo)
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

func createPostgresTestContainer(ctx context.Context, schemaName, dbUser, dbUserPwd string) (ctr *pgtc.PostgresContainer, err error) {

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

func createPostgresDBInstance(ctx context.Context, container testcontainers.Container, schemaName, dbUser, dbUserPwd string) (*sql.DB, error) {
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
