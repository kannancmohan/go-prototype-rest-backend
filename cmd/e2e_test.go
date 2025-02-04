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
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	api_app "github.com/kannancmohan/go-prototype-rest-backend/cmd/api/app"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/testutils"
	tc_testutils "github.com/kannancmohan/go-prototype-rest-backend/internal/testutils/testcontainers"
	"github.com/redis/go-redis/v9"
)

var (
	serverAddr string
)

func TestMain(m *testing.M) {
	sysStopChan := app_common.SysInterruptStopChan()

	setup := testutils.NewInfraSetup(
		testutils.NewInfraSetupFuncRegistry("postgres", setupTestPostgres),
		testutils.NewInfraSetupFuncRegistry("redis", setupTestRedis),
		testutils.NewInfraSetupFuncRegistry("elasticsearch", setupTestElasticsearch),
		testutils.NewInfraSetupFuncRegistry("kafka", setupTestKafka),
	)
	go setup.Start()

	var exitCode int
	select {
	case <-setup.SetupDoneChan:
		if err := initApp(sysStopChan); err != nil {
			log.Printf("App init failed with error:%s", err.Error())
			exitCode = 1
		} else {
			log.Println("Setup and App init done, executing test cases..")
			exitCode = m.Run()
		}
	case <-setup.SetupErrChan:
		log.Println("Setup failed with error")
		exitCode = 1
	case <-sysStopChan:
		log.Println("Received interrupt signal")
		exitCode = 1
	}

	//TODO - handle proper cleanup on interrupt signal. i.e terminating test in middle of container creation
	setup.Cleanup(context.Background())
	os.Exit(exitCode)
}

func TestUserEndpoints(t *testing.T) {
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

// func TestPostsEndpoints(t *testing.T) {
// 	prerequisiteTC, err := testutils.LoadTestCases("./e2e_testdata/posts/test_case_prerequisites.json")
// 	if err != nil || len(prerequisiteTC) < 1 {
// 		t.Fatalf("failed to load test cases: %v", err)
// 	}
// 	createTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_create_user.json")
// 	if err != nil {
// 		t.Fatalf("failed to load test cases: %v", err)
// 	}
// 	// updateTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_update_user.json")
// 	// if err != nil {
// 	// 	t.Fatalf("failed to load test cases: %v", err)
// 	// }
// 	// getTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_get_user.json")
// 	// if err != nil {
// 	// 	t.Fatalf("failed to load test cases: %v", err)
// 	// }
// 	// deleteTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_delete_user.json")
// 	// if err != nil {
// 	// 	t.Fatalf("failed to load test cases: %v", err)
// 	// }
// 	client := &http.Client{}
// 	// create a prerequisite test user
// 	resp, err := sendRequest(serverAddr, client, prerequisiteTC[0].Request)
// 	if err != nil {
// 		t.Fatalf("failed to send request for test:%s. received (error: %v)", tc.Name, err)
// 	}
// 	testcases := slices.Concat(prerequisiteTC, createTC)

// }

func setupTestPostgres(ctx context.Context) (testutils.InfraSetupCleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	instance := tc_testutils.NewTestPostgresContainer("e2e_test", "test", "test")
	container, cleanupFunc, err := instance.CreatePostgresTestContainer()
	if err != nil {
		return cleanupFunc, err
	}

	db, err := instance.CreatePostgresDBInstance(container)
	if err != nil {
		return cleanupFunc, err
	}

	if err := tc_testutils.ApplyDBMigrations(db); err != nil {
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

func setupTestRedis(ctx context.Context) (testutils.InfraSetupCleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	instance := tc_testutils.NewTestRedisContainer()
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

func setupTestElasticsearch(ctx context.Context) (testutils.InfraSetupCleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	instance := tc_testutils.NewTestElasticsearchContainer()
	container, cleanupFunc, err := instance.CreateElasticsearchTestContainer("")
	if err != nil {
		return cleanupFunc, err
	}

	addr := container.Settings.Address

	os.Setenv("ELASTIC_HOST", addr)
	os.Setenv("ELASTIC_POST_INDEX_NAME", "e2e_test_posts")
	return cleanupFunc, nil
}

func setupTestKafka(ctx context.Context) (testutils.InfraSetupCleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	instance := tc_testutils.NewTestKafkaContainer("e2e-test-kafka")
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

func getResponseBody[T any](resp *http.Response) (T, error) {
	var respBody T
	if resp == nil || resp.Body == nil {
		return respBody, fmt.Errorf("http response/body is nil")
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return respBody, fmt.Errorf("error decoding response body.%w", err)
	}
	return respBody, nil
}

func validateResponse[T any](t *testing.T, resp *http.Response, expected *testutils.HttpTestExpectedResp, validateBody func(T, T) error) {
	if expected.Status != resp.StatusCode {
		t.Errorf("Expected statuscode:%d, but instead received statuscode:%d", expected.Status, resp.StatusCode)
	}
	if expected.Body != nil {
		respBody, err := getResponseBody[T](resp)
		if err != nil {
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

func initApp(stopChan ...app_common.StopChan) error {
	port, err := testutils.GetFreePort()
	if err != nil {
		return fmt.Errorf("failed to get a free port: %w", err)
	}
	os.Setenv("API_PORT", port)

	// Start the application in a goroutine
	errChan := make(chan error, 2)
	go func() {
		err := api_app.ListenAndServe("", stopChan...)
		if err != nil {
			errChan <- err
			return
		}
	}()
	go func() {
		if err := testutils.WaitForPort(port, 10*time.Second); err != nil {
			errChan <- err
		}
		close(errChan)
	}()

	err = <-errChan // block until errChan is triggered
	if err != nil {
		return err
	}

	serverAddr = fmt.Sprintf("http://localhost:%s", port)
	return nil
}
