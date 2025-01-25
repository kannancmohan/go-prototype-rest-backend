package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/testutils"
	"github.com/redis/go-redis/v9"
)

func TestMain(m *testing.M) {

	setupCtx, setupCancel := context.WithCancel(context.Background())
	defer setupCancel()

	var cleanupFuncs = make(map[string]func(context.Context) error) //map to hold cleanup functions of setup

	// Helper to register setup and cleanup
	setup := func(name string, setupFunc func() (func(context.Context) error, error)) {

		if setupCtx.Err() != nil { // Skip setup if the setupCtx is already canceled
			log.Printf("Skipping setup for %s due to previous failure.", name)
			return
		}

		cleanupFunc, err := setupFunc() // invoke the actual func
		if cleanupFunc != nil {
			cleanupFuncs[name] = cleanupFunc
		}
		if err != nil {
			log.Printf("Failed to start %s testcontainer: %v", name, err)
			setupCancel() // manually call cancel setupCtx so as to stop executing further
			return
		}
	}

	//TODO add goroutine to run asynchronously
	setup("postgres", setupTestPostgres)
	setup("redis", setupTestRedis)
	setup("elasticsearch", setupTestElasticsearch)
	setup("kafka", setupTestKafka)

	if setupCtx.Err() != nil { // true if setupCancel was called during setup
		log.Println("Setup failed. Skipping tests.")
		runCleanup(cleanupFuncs)
		os.Exit(1) // Exit with a failure code
	}

	code := m.Run()
	runCleanup(cleanupFuncs)

	os.Exit(code)

}

func TestRoleStore_XXX(t *testing.T) {
	port, _ := testutils.GetFreePort()
	os.Setenv("API_PORT", port)
	apiCmd := exec.Command("go", "run", "./cmd/api/")
	if out, err := apiCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to start API service: %v\n%s", err, string(out))
	}

	// apiCmd.Stdout = os.Stdout
	// apiCmd.Stderr = os.Stderr
	// apiCmd.Env = os.Environ() // Use environment variables set in TestMain
	// if err := apiCmd.Start(); err != nil {
	// 	t.Fatalf("Failed to start API service: %v", err)
	// }
	defer apiCmd.Process.Kill()
}

func setupTestPostgres() (func(ctx context.Context) error, error) {
	instance := testutils.NewTestPostgresContainer("e2e_test", "test", "test")
	container, cleanupFunc, err := instance.CreatePostgresTestContainer()
	if err != nil {
		return cleanupFunc, err
	}

	db, err := instance.CreatePostgresDBInstance(container)
	if err != nil {
		return cleanupFunc, err
	}

	if err := testutils.ApplyDBMigrations(db); err != nil {
		return cleanupFunc, err
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	host, err := container.Host(timeoutCtx)
	if err != nil {
		return cleanupFunc, err
	}

	port, err := container.MappedPort(timeoutCtx, nat.Port("5432/tcp"))
	if err != nil {
		return cleanupFunc, err
	}

	os.Setenv("DB_HOST", host)
	os.Setenv("DB_PORT", port.Port())
	os.Setenv("DB_USER", "test")
	os.Setenv("DB_PASS", "test")
	os.Setenv("DB_SSL_MODE", "disable")
	os.Setenv("API_DB_SCHEMA_NAME", "e2e_test")
	return cleanupFunc, nil
}

func setupTestRedis() (func(ctx context.Context) error, error) {
	instance := testutils.NewTestRedisContainer()
	container, cleanupFunc, err := instance.CreateRedisTestContainer("")
	if err != nil {
		return cleanupFunc, err
	}

	connStr, err := instance.GetRedisConnectionString(container)
	if err != nil {
		return cleanupFunc, err
	}

	connOpt, err := redis.ParseURL(connStr)
	if err != nil {
		return nil, err
	}

	os.Setenv("REDIS_HOST", connOpt.Addr)
	os.Setenv("REDIS_DB", strconv.Itoa(connOpt.DB))
	return cleanupFunc, nil
}

func setupTestElasticsearch() (func(ctx context.Context) error, error) {
	instance := testutils.NewTestElasticsearchContainer()
	container, cleanupFunc, err := instance.CreateElasticsearchTestContainer("")
	if err != nil {
		return cleanupFunc, err
	}

	addr := container.Settings.Address

	os.Setenv("ELASTIC_HOST", addr)
	os.Setenv("ELASTIC_POST_INDEX_NAME", "e2e_test_posts")
	return cleanupFunc, nil
}

func setupTestKafka() (func(ctx context.Context) error, error) {
	instance := testutils.NewTestKafkaContainer("e2e-test-kafka")
	container, cleanupFunc, err := instance.CreateKafkaTestContainer()
	if err != nil {
		return cleanupFunc, err
	}

	addr, err := instance.GetKafkaBrokerAddress(container)
	if err != nil {
		return cleanupFunc, err
	}

	os.Setenv("KAFKA_HOST", addr)
	os.Setenv("API_KAFKA_TOPIC", "e2e_test_posts")
	return cleanupFunc, nil
}

func runCleanup(cleanupFuncs map[string]func(context.Context) error) {
	ctx := context.Background()
	for key, cleanup := range cleanupFuncs {
		log.Printf("Cleanup triggered for: %s", key)
		if err := cleanup(ctx); err != nil {
			log.Printf("Cleanup error for %s: %v", key, err)
		}
	}
}
