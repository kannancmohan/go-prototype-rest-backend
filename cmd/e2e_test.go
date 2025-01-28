//go:build !skip_docker_tests

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/testutils"
	"github.com/redis/go-redis/v9"
)

func TestMain(m *testing.M) {
	cleanupRegistry := NewCleanupRegistry()

	var setupWG sync.WaitGroup
	setupErrChan := make(chan error, 4) // Buffer size equals the number of services

	//TODO - handle proper cleanup on terminating test in middle of container creation
	go handleInterruptSignals(cleanupRegistry) // do cleanup on interrupt signals

	// Helper to register setup and cleanup
	setup := func(name string, setupFunc func() (CleanupFunc, error)) {
		setupWG.Add(1)
		go func(name string, setup func() (CleanupFunc, error)) {
			defer setupWG.Done()
			cleanup, err := setup()
			if cleanup != nil {
				cleanupRegistry.register(name, cleanup)
			}
			if err != nil {
				log.Printf("Failed to start %s testcontainer: %v", name, err)
				setupErrChan <- err
				return
			}
			log.Printf("Successfully started %s testconatiner", name)
		}(name, setupFunc)
	}

	setup("postgres", setupTestPostgres)
	setup("redis", setupTestRedis)
	setup("elasticsearch", setupTestElasticsearch)
	setup("kafka", setupTestKafka)

	setupWG.Wait()

	// Check if any service failed
	if len(setupErrChan) > 0 {
		for len(setupErrChan) > 0 {
			err := <-setupErrChan
			log.Printf("Setup failed with error: %v", err)
		}
		cleanupRegistry.runCleanup(context.Background())
		os.Exit(1)
	}

	code := runTestsWithTimeout(m, cleanupRegistry)
	os.Exit(code)
}

func TestUserEndpoints(t *testing.T) {
	port, _ := testutils.GetFreePort()
	os.Setenv("API_PORT", port)
	apiCmd := exec.Command("go", "run", "./api/")
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

	createTC, err := testutils.LoadTestCases("./e2e_testdata/test_case_create_user.json")
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	updateTC, err := testutils.LoadTestCases("./e2e_testdata/test_case_update_user.json")
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	getTC, err := testutils.LoadTestCases("./e2e_testdata/test_case_get_user.json")
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	testcases := slices.Concat(createTC, updateTC, getTC)

	serverAddr := fmt.Sprintf("http://localhost:%s", port)
	client := &http.Client{}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := sendRequest(serverAddr, client, tc.Request)
			if err != nil {
				t.Fatalf("failed to send request for test:%s. received (error: %v)", tc.Name, err)
			}
			defer resp.Body.Close()
			validateResponse(t, resp, &tc.Expected, compareTestUser)
		})
	}
}

func setupTestPostgres() (CleanupFunc, error) {
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

func setupTestRedis() (CleanupFunc, error) {
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

func setupTestElasticsearch() (CleanupFunc, error) {
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

func setupTestKafka() (CleanupFunc, error) {
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

func handleInterruptSignals(cleanupRegistry *cleanupRegistry) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigChan
	log.Println("Received interrupt signal. Running cleanup...")
	cleanupRegistry.runCleanup(context.Background())
	os.Exit(1)
}

func runTestsWithTimeout(m *testing.M, cleanupRegistry *cleanupRegistry) int {
	resultChan := make(chan int)
	go func() {
		resultChan <- m.Run()
	}()

	timeout := 30 * time.Second // Set a timeout for the tests
	select {
	case code := <-resultChan: // Tests completed successfully
		cleanupRegistry.runCleanup(context.Background())
		return code
	case <-time.After(timeout): // Test timed out
		log.Printf("Test timed out after %v. Running cleanup...\n", timeout)
		cleanupRegistry.runCleanup(context.Background())
		return 1
	}
}

type CleanupFunc func(context.Context) error

type cleanupRegistry struct {
	mu    sync.Mutex
	funcs map[string]CleanupFunc
}

func NewCleanupRegistry() *cleanupRegistry {
	return &cleanupRegistry{
		funcs: make(map[string]CleanupFunc),
	}
}

func (r *cleanupRegistry) register(name string, cleanup CleanupFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.funcs[name] = cleanup
}

func (r *cleanupRegistry) runCleanup(ctx context.Context) {
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

func convertExpectedBody[T any](data any) (T, error) {
	var v T
	jsonData, err := json.Marshal(data)
	if err != nil {
		return v, fmt.Errorf("Error marshalling map to JSON: %s", err.Error())
	}

	err = json.Unmarshal(jsonData, &v)
	if err != nil {
		return v, fmt.Errorf("Error unmarshalling JSON into struct: %s", err.Error())
	}
	return v, nil
}

func sendRequest(serverAddr string, client *http.Client, testReq testutils.HttpTestRequest) (*http.Response, error) {
	body, err := json.Marshal(testReq.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal test request body: %w", err)
	}
	req, err := http.NewRequest(testReq.Method, serverAddr+testReq.Path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range testReq.Headers {
		req.Header.Set(key, value)
	}
	return client.Do(req)
}

func validateResponse[T any](t *testing.T, resp *http.Response, expected *testutils.HttpTestExpectedResp, validateBody func(T, T) error) {
	if expected.Status != resp.StatusCode {
		t.Errorf("Expected statuscode:%d, but instead received statuscode:%d", expected.Status, resp.StatusCode)
	}
	if expected.Body != nil {
		var respBody T
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			t.Errorf("error decoding response body. received (error: %v)", err)
		}
		expBody, err := convertExpectedBody[T](expected.Body)
		if err != nil {
			t.Errorf("error decoding expected body received (error: %v)", err)
		}

		if err := validateBody(expBody, respBody); err != nil {
			t.Error(err.Error())
		}
	}
	if expected.Error != nil {
		var respBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			t.Errorf("error decoding response body. received (error: %v)", err)
		}
		if !reflect.DeepEqual(expected.Error, respBody) {
			t.Errorf("Expected error:%v, but received %v instead.", expected.Error, respBody)
		}
	}
}

func compareTestUser(expected, actual model.User) error {
	var isUserNameInvalid = (expected.Username != "" && actual.Username != expected.Username)
	var isEmailInvalid = (expected.Email != "" && actual.Email != expected.Email)
	if actual.ID < 1 || isUserNameInvalid || isEmailInvalid {
		return fmt.Errorf("Expected response body:%v,instead received body:%v", expected, actual)
	}
	return nil
}
