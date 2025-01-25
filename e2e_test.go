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
	"golang.org/x/sync/errgroup"
)

func TestMain(m *testing.M) {

	var cleanupFuncs = make(map[string]func(context.Context) error) //map to hold cleanup functions of setup
	var mu sync.Mutex                                               // Protects cleanupFuncs

	// Catch interrupt signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT,os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
    go func() {
        <-sigChan
        log.Println("Received interrupt signal. Running cleanup...")
        runCleanup(cleanupFuncs)
        os.Exit(1)
    }()

	g, ctx := errgroup.WithContext(context.Background()) // Create an errgroup to manage group of goroutines that can fail and cancel each other

	// Helper to register setup and cleanup
	setup := func(name string, setupFunc func() (func(context.Context) error, error)) {
		g.Go(func() error {
			if ctx.Err() != nil { // Check if the context has been canceled (e.g., due to a failure in another setup)
				log.Printf("Skipping setup for %s due to cancellation.", name)
				return nil
			}
			cleanupFunc, err := setupFunc() // Run the setup function
			if cleanupFunc != nil {         // Register the cleanup function if not nil
				mu.Lock()
				cleanupFuncs[name] = cleanupFunc
				mu.Unlock()
			}
			if err != nil {
				log.Printf("Failed to start %s testcontainer: %v", name, err)
				return err // Return error to cancel the group
			}
			log.Printf("Successfully started %s testconatiner", name)
			return nil
		})
	}

	setup("postgres", setupTestPostgres)
	setup("redis", setupTestRedis)
	setup("elasticsearch", setupTestElasticsearch)
	setup("kafka", setupTestKafka)

	if err := g.Wait(); err != nil { // Wait for all setup goroutines to complete
		log.Println("Setup failed. Skipping tests.")
		runCleanup(cleanupFuncs)
		os.Exit(1)
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
	var wg sync.WaitGroup
	var mu sync.Mutex
	var cleanupErrors []error

	for name, cleanup := range cleanupFuncs {
		wg.Add(1)
		go func(name string, cleanup func(context.Context) error) {
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
