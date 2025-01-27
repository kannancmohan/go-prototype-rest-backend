package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/testutils"
	"github.com/redis/go-redis/v9"
)

func TestMain(m *testing.M) {
	cleanupRegistry := NewCleanupRegistry()

	// do cleanup in interrupt signals
	go handleInterruptSignals(cleanupRegistry)

	var wg sync.WaitGroup
	errChan := make(chan error, 4) // Buffer size equals the number of services

	// Helper to register setup and cleanup
	setup := func(name string, setupFunc func() (func(context.Context) error, error)) {
		wg.Add(1)
		go func(name string, setup func() (func(context.Context) error, error)) {
			defer wg.Done()
			cleanup, err := setup()
			if cleanup != nil {
				cleanupRegistry.Register(name, cleanup)
			}
			if err != nil {
				log.Printf("Failed to start %s testcontainer: %v", name, err)
				errChan <- err
				return
			}
			log.Printf("Successfully started %s testconatiner", name)
		}(name, setupFunc)
	}

	setup("postgres", setupTestPostgres)
	setup("redis", setupTestRedis)
	setup("elasticsearch", setupTestElasticsearch)
	setup("kafka", setupTestKafka)

	wg.Wait()

	// Check if any service failed
	select {
	case err := <-errChan:
		log.Printf("setup failed with error:%v", err)
		cleanupRegistry.RunCleanup(context.Background())
		os.Exit(1)
	default:
		//runCleanup(cleanupFuncs)
	}

	code := runTestsWithTimeout(m, cleanupRegistry)
	os.Exit(code)
}

func TestRoleStore_XXX(t *testing.T) {
	port, _ := testutils.GetFreePort()
	os.Setenv("API_PORT", port)
	apiCmd := exec.Command("go", "run", "./cmd/api/")
	if err := apiCmd.Start(); err != nil {
		t.Fatalf("Failed to start API service: %v", err)
	}

	defer func() {
		if err := apiCmd.Process.Kill(); err != nil {
			t.Logf("Failed to kill API service process: %v", err)
		}
	}()

	if err := testutils.WaitForPort(port, 10*time.Second); err != nil {
		t.Fatalf("Server did not start: %v", err)
	}

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
	//return nil, fmt.Errorf("test error")
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

func handleInterruptSignals(cleanupRegistry *CleanupRegistry) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigChan
	log.Println("Received interrupt signal. Running cleanup...")
	cleanupRegistry.RunCleanup(context.Background())
	os.Exit(1)
}

func runTestsWithTimeout(m *testing.M, cleanupRegistry *CleanupRegistry) int {
	resultChan := make(chan int)
	go func() {
		resultChan <- m.Run()
	}()

	timeout := 30 * time.Second // Set a timeout for the tests
	select {
	case code := <-resultChan: // Tests completed successfully
		cleanupRegistry.RunCleanup(context.Background())
		return code
	case <-time.After(timeout): // Test timed out
		log.Printf("Test timed out after %v. Running cleanup...\n", timeout)
		cleanupRegistry.RunCleanup(context.Background())
		return 1
	}
}

type CleanupFunc func(context.Context) error

type CleanupRegistry struct {
	mu    sync.Mutex
	funcs map[string]CleanupFunc
}

func NewCleanupRegistry() *CleanupRegistry {
	return &CleanupRegistry{
		funcs: make(map[string]CleanupFunc),
	}
}

func (r *CleanupRegistry) Register(name string, cleanup CleanupFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.funcs[name] = cleanup
}

func (r *CleanupRegistry) RunCleanup(ctx context.Context) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var cleanupErrors []error

	for name, cleanup := range r.funcs {
		wg.Add(1)
		go func(name string, cleanup CleanupFunc) {
			defer wg.Done()
			if err := cleanup(ctx); err != nil {
				mu.Lock()
				cleanupErrors = append(cleanupErrors, err)
				log.Printf("Failed to cleanup %s: %v", name, err)
				mu.Unlock()
			} else {
				log.Printf("Successfully cleaned up %s", name)
			}
		}(name, cleanup)
	}

	wg.Wait()

	if len(cleanupErrors) > 0 {
		log.Println("Cleanup completed with errors.")
	}
}
