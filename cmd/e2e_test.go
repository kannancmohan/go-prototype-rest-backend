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
	interruptSigChan := make(chan os.Signal, 1) // Interrupt signal channel
	go func() {
		signal.Notify(interruptSigChan, syscall.SIGINT, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	}()

	setup := NewSetup(
		NewSetupFuncRegistry("postgres", setupTestPostgres),
		NewSetupFuncRegistry("redis", setupTestRedis),
		NewSetupFuncRegistry("elasticsearch", setupTestElasticsearch),
		NewSetupFuncRegistry("kafka", setupTestKafka),
	)
	go setup.start()

	var exitCode int
	select {
	case <-setup.doneC:
		log.Println("Setup done. Executing test cases...")
		exitCode = m.Run()
	case <-setup.errorC:
		log.Println("Setup failed with error")
		exitCode = 1
	case <-interruptSigChan:
		log.Println("Received interrupt signal")
		exitCode = 1
	}

	//TODO - handle proper cleanup on interrupt signal. i.e terminating test in middle of container creation
	setup.cleanup(context.Background())
	os.Exit(exitCode)
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

	createTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_create_user.json")
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	updateTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_update_user.json")
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	getTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_get_user.json")
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	deleteTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_delete_user.json")
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	testcases := slices.Concat(createTC, updateTC, getTC, deleteTC)

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

func setupTestPostgres(ctx context.Context) (CleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
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

func setupTestRedis(ctx context.Context) (CleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
		return cleanupFunc, err
	}

	os.Setenv("REDIS_HOST", connOpt.Addr)
	os.Setenv("REDIS_DB", strconv.Itoa(connOpt.DB))
	return cleanupFunc, nil
}

func setupTestElasticsearch(ctx context.Context) (CleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

func setupTestKafka(ctx context.Context) (CleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

type SetupFunc func(context.Context) (CleanupFunc, error)
type CleanupFunc func(context.Context) error

type setup struct {
	setupFuncRegistries []*setupFuncRegistry
	doneC               chan struct{} // to signal setup has done
	errorC              chan struct{} // to signal setup has error
}

func NewSetup(setupFuncRegistries ...*setupFuncRegistry) *setup {
	return &setup{setupFuncRegistries: setupFuncRegistries, doneC: make(chan struct{}), errorC: make(chan struct{})}
}

func (s *setup) start() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	var setupWG sync.WaitGroup
	for _, setupReg := range s.setupFuncRegistries {
		setupWG.Add(1)
		go func(name string, setupFunc SetupFunc) {
			defer setupWG.Done()
			cleanup, err := setupFunc(ctx)
			setupReg.addCleanupFunc(cleanup)
			if err != nil {
				log.Printf("Failed to start %s container: %v", name, err)
				cancel()        // Cancel the context so as other setup could be stopped
				close(s.errorC) // notify that there was error in setup
				return          // return from goroutine
			}
			log.Printf("Successfully started %s container", name)
		}(setupReg.name, setupReg.setupFunc)
	}
	setupWG.Wait()
	close(s.doneC) // notify that setup has been done
}

func (s *setup) cleanup(ctx context.Context) {
	log.Print("Invoking cleanup...")
	var cleanupWG sync.WaitGroup
	for _, setupReg := range s.setupFuncRegistries {
		cleanupWG.Add(1)
		go func(setupReg *setupFuncRegistry) {
			defer cleanupWG.Done()
			if setupReg.cleanupFunc != nil {
				if err := setupReg.cleanupFunc(ctx); err != nil {
					log.Printf("Failed to cleanup %s: %v", setupReg.name, err)
					return // return from goroutine
				}
				log.Printf("Successfully cleaned up %s", setupReg.name)
			}
		}(setupReg)
	}
	cleanupWG.Wait()
}

type setupFuncRegistry struct {
	name        string
	setupFunc   SetupFunc
	cleanupFunc CleanupFunc
}

func NewSetupFuncRegistry(name string, setupFunc SetupFunc) *setupFuncRegistry {
	return &setupFuncRegistry{
		name:      name,
		setupFunc: setupFunc,
	}
}

func (s *setupFuncRegistry) addCleanupFunc(cleanupFunc CleanupFunc) {
	s.cleanupFunc = cleanupFunc
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
