//go:build !skip_docker_tests

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	api_app "github.com/kannancmohan/go-prototype-rest-backend/cmd/api/app"
	search_indexer_app "github.com/kannancmohan/go-prototype-rest-backend/cmd/elasticsearch-indexer-kafka/app"
	app_common "github.com/kannancmohan/go-prototype-rest-backend/cmd/internal/common"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/model"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/common/domain/store"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/testutils"
	tc_testutils "github.com/kannancmohan/go-prototype-rest-backend/internal/testutils/testcontainers"
	"github.com/redis/go-redis/v9"
)

const (
	kafkaTopic               = "e2e_test_posts"
	kafkaConsumerGroupId     = "e2e-elasticsearch-indexer"
	elasticsearchIndexerName = "posts"
	dbSchemaName             = "e2e_test"
	dbUserName               = "test"
	dbUserPwd                = "test"
)

var apiServerAddr string

func TestMain(m *testing.M) {
	os.Exit(doTest(m))
}

func TestUserEndpoints(t *testing.T) {
	client := &http.Client{}

	// create prerequisite user required for testing update,get and delete endpoints
	user, err := sendAndGetResponseBody[model.User](apiServerAddr, client, testutils.HttpTestRequest{
		Method:  "POST",
		Path:    "/api/v1/authentication/user",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    map[string]string{"username": "e2e_usertest_user01", "email": "e2e_usertest_user01@test.com", "password": "e2e_usertest_user01", "role": "admin"},
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	createTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_create_user.json", nil)
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	updateTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_update_user.json", map[string]interface{}{"userID": user.ID})
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	getTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_get_user.json", map[string]interface{}{"userID": user.ID, "userEmail": user.Email})
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	deleteTC, err := testutils.LoadTestCases("./e2e_testdata/user/test_case_delete_user.json", map[string]interface{}{"userID": user.ID})
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}

	testcases := slices.Concat(createTC, updateTC, getTC, deleteTC)

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := testutils.SendRequest(apiServerAddr, client, tc.Request)
			if err != nil {
				t.Fatalf("failed to send request for test:%s. received (error: %v)", tc.Name, err)
			}
			defer resp.Body.Close()
			validateResponse(t, resp, &tc.Expected, compareTestUser)
		})
	}
}

func TestPostsEndpoints(t *testing.T) {
	client := &http.Client{}

	// create prerequisite user before creating posts
	user, err := sendAndGetResponseBody[model.User](apiServerAddr, client, testutils.HttpTestRequest{
		Method:  "POST",
		Path:    "/api/v1/authentication/user",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    map[string]string{"username": "e2e_poststest_user01", "email": "e2e_poststest_user01@test.com", "password": "e2e_poststest_user01", "role": "admin"},
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// create prerequisite post required for testing update,get and delete endpoints
	post, err := sendAndGetResponseBody[model.Post](apiServerAddr, client, testutils.HttpTestRequest{
		Method:  "POST",
		Path:    "/api/v1/posts",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    map[string]interface{}{"user_id": user.ID, "title": "e2e test title01", "content": "e2e test content01", "tags": []string{"e2e_test", "test01"}},
	})
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	createTC, err := testutils.LoadTestCases("./e2e_testdata/posts/test_case_create_posts.json", map[string]interface{}{"userID": user.ID})
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	updateTC, err := testutils.LoadTestCases("./e2e_testdata/posts/test_case_update_posts.json", map[string]interface{}{"postID": post.ID})
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	getTC, err := testutils.LoadTestCases("./e2e_testdata/posts/test_case_get_posts.json", map[string]interface{}{"postID": post.ID})
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}
	deleteTC, err := testutils.LoadTestCases("./e2e_testdata/posts/test_case_delete_posts.json", map[string]interface{}{"postID": post.ID})
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}

	testcases := slices.Concat(createTC, updateTC, getTC, deleteTC)

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := testutils.SendRequest(apiServerAddr, client, tc.Request)
			if err != nil {
				t.Fatalf("failed to send request for test:%s. received (error: %v)", tc.Name, err)
			}
			defer resp.Body.Close()
			validateResponse(t, resp, &tc.Expected, compareTestUser)
		})
	}

}

func TestSearchPostsEndpoints(t *testing.T) {
	client := &http.Client{}

	// create prerequisite user before creating posts
	user, err := sendAndGetResponseBody[model.User](apiServerAddr, client, testutils.HttpTestRequest{
		Method:  "POST",
		Path:    "/api/v1/authentication/user",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    map[string]string{"username": "e2e_searchpoststest_user01", "email": "e2e_searchpoststest_user01@test.com", "password": "e2e_searchpoststest_user01", "role": "admin"},
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// create prerequisite post required for testing search endpoint
	post, err := sendAndGetResponseBody[model.Post](apiServerAddr, client, testutils.HttpTestRequest{
		Method:  "POST",
		Path:    "/api/v1/posts",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    map[string]interface{}{"user_id": user.ID, "title": "searchposttest_title", "content": "searchposttest_content", "tags": []string{"e2e_test", "searchposttest01"}},
	})
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	searchTC, err := testutils.LoadTestCases("./e2e_testdata/posts/test_case_search_posts.json", map[string]interface{}{"userID": user.ID, "postID": post.ID})
	if err != nil {
		t.Fatalf("failed to load test cases: %v", err)
	}

	time.Sleep(3 * time.Second) // wait for few seconds so that the above post could be consumed by kafka and indexed to elasticsearch

	for _, tc := range searchTC {
		t.Run(tc.Name, func(t *testing.T) {
			resp, err := testutils.SendRequest(apiServerAddr, client, tc.Request)
			if err != nil {
				t.Fatalf("failed to send request for test:%s. received (error: %v)", tc.Name, err)
			}
			defer resp.Body.Close()
			validateResponse(t, resp, &tc.Expected, comparePostSearchResponse)
		})
	}

}

func doTest(m *testing.M) (exitCode int) {
	sysStopChan := app_common.SysInterruptStopChan()        // Create a chan to capture os interrupt signal
	ctx, cancel := context.WithCancel(context.Background()) // Create a cancelable context
	defer cancel()

	defer func() { // Recover from panics and log the stack trace
		if r := recover(); r != nil {
			stackTrace := string(debug.Stack()) // Capture the stack trace
			log.Printf("Recovered from panic:%v stack_trace[%s]", r, stackTrace)
		}
	}()

	// cancels the ctx on receiving os-interrupt signal. cancelling the ctx should trigger the context aware logics
	go func() {
		<-sysStopChan
		fmt.Println("Received OS interrupt signal. Cancelling main context...")
		cancel()
	}()

	setup := testutils.NewInfraSetup(
		testutils.NewInfraSetupFuncRegistry("postgres", setupTestInstancePostgres),
		testutils.NewInfraSetupFuncRegistry("redis", setupTestInstanceRedis),
		testutils.NewInfraSetupFuncRegistry("elasticsearch", setupTestInstanceElasticsearch),
		testutils.NewInfraSetupFuncRegistry("kafka", setupTestInstanceKafka),
	)
	defer setup.Cleanup(context.Background())

	go setup.Start(ctx)

	//wait for setup to be done or cancelled
	select {
	case <-ctx.Done():
		log.Println("Context cancelled. Cancelling setup...")
		return 1
	case <-setup.SetupErrChan:
		log.Println("Setup failed with error")
		return 1
	case <-setup.SetupDoneChan:
		log.Println("Setup done..")
	}

	apps := testutils.NewAppSetup().WithAddSetupFunc("api", startApiApp).WithAddSetupFunc("search-indexer", startSearchIndexerApp)

	go apps.Start(ctx)

	//wait for apps to start or cancel
	select {
	case <-ctx.Done():
		log.Println("Context cancelled. Cancelling Apps...")
		return 1
	case <-apps.ErrChan:
		log.Println("Apps failed with error")
		return 1
	case <-apps.DoneChan:
		log.Println("Apps started.., Executing test cases")
	}

	//get the server address for api app
	apiAppResp, err := apps.GetAppSetupFunResponse("api")
	if err != nil {
		log.Println("failed - missing required api app server address in appsetup response")
		return 1
	}
	apiServerAddr = apiAppResp.Addr

	// run the test
	return m.Run()
}

func setupTestInstancePostgres(ctx context.Context) (testutils.InfraSetupCleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	instance := tc_testutils.NewTestPostgresContainer(dbSchemaName, dbUserName, dbUserPwd)
	container, cntCleanupFunc, err := instance.CreatePostgresTestContainer()
	if err != nil {
		return cntCleanupFunc, err
	}

	db, err := instance.CreatePostgresDBInstance(container)
	if err != nil {
		return cntCleanupFunc, err
	}

	if err := tc_testutils.ApplyDBMigrations(db); err != nil {
		return cntCleanupFunc, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	host, err := container.Host(timeoutCtx)
	if err != nil {
		return cntCleanupFunc, err
	}

	port, err := container.MappedPort(timeoutCtx, nat.Port("5432/tcp"))
	if err != nil {
		return cntCleanupFunc, err
	}

	os.Setenv("DB_HOST", host)
	os.Setenv("DB_PORT", port.Port())
	os.Setenv("DB_USER", dbUserName)
	os.Setenv("DB_PASS", dbUserPwd)
	os.Setenv("DB_SSL_MODE", "disable")

	cleanupFunc := func(ctx context.Context) error {
		err := cntCleanupFunc(ctx)
		if err != nil {
			return err
		}
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASS")
		os.Unsetenv("DB_SSL_MODE")
		return nil
	}
	return cleanupFunc, nil
}

func setupTestInstanceRedis(ctx context.Context) (testutils.InfraSetupCleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	instance := tc_testutils.NewTestRedisContainer()
	container, cntCleanupFunc, err := instance.CreateRedisTestContainer("")
	if err != nil {
		return cntCleanupFunc, err
	}

	connStr, err := instance.GetRedisConnectionString(container)
	if err != nil {
		return cntCleanupFunc, err
	}

	connOpt, err := redis.ParseURL(connStr)
	if err != nil {
		return cntCleanupFunc, err
	}

	os.Setenv("REDIS_HOST", connOpt.Addr)
	os.Setenv("REDIS_DB", strconv.Itoa(connOpt.DB))

	cleanupFunc := func(ctx context.Context) error {
		err := cntCleanupFunc(ctx)
		if err != nil {
			return err
		}
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_DB")
		return nil
	}
	return cleanupFunc, nil
}

func setupTestInstanceElasticsearch(ctx context.Context) (testutils.InfraSetupCleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	instance := tc_testutils.NewTestElasticsearchContainer()
	indexMappings := map[string]interface{}{
		elasticsearchIndexerName: map[string]interface{}{
			"properties": map[string]interface{}{
				"created_at": map[string]interface{}{"type": "date", "ignore_malformed": true},
			},
		},
	}
	container, cntCleanupFunc, err := instance.CreateElasticsearchTestContainer("", indexMappings)
	if err != nil {
		return cntCleanupFunc, err
	}

	addr := container.Settings.Address

	os.Setenv("ELASTIC_HOST", addr)
	os.Setenv("ELASTIC_POST_INDEX_NAME", elasticsearchIndexerName)

	cleanupFunc := func(ctx context.Context) error {
		err := cntCleanupFunc(ctx)
		if err != nil {
			return err
		}
		os.Unsetenv("ELASTIC_HOST")
		os.Unsetenv("ELASTIC_POST_INDEX_NAME")
		return nil
	}
	return cleanupFunc, nil
}

func setupTestInstanceKafka(ctx context.Context) (testutils.InfraSetupCleanupFunc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	instance := tc_testutils.NewTestKafkaContainer("e2e-test-kafka")
	container, cntCleanupFunc, err := instance.CreateKafkaTestContainer(ctx)
	if err != nil {
		return cntCleanupFunc, err
	}

	addresses, err := container.Brokers(ctx)
	if err != nil {
		return cntCleanupFunc, err
	}

	os.Setenv("KAFKA_HOST", addresses[0])

	cleanupFunc := func(ctx context.Context) error {
		err := cntCleanupFunc(ctx)
		if err != nil {
			return err
		}
		os.Unsetenv("KAFKA_HOST")
		return nil
	}
	return cleanupFunc, nil
}

func startApiApp(ctx context.Context) (testutils.AppSetupFuncResponse, error) {
	var appFuncResponse testutils.AppSetupFuncResponse
	if err := ctx.Err(); err != nil {
		return appFuncResponse, err
	}

	port, err := testutils.GetFreePort()
	if err != nil {
		return appFuncResponse, fmt.Errorf("failed to get a free port: %w", err)
	}
	os.Setenv("APP_API_PORT", port)
	os.Setenv("APP_API_KAFKA_TOPIC", kafkaTopic)
	os.Setenv("APP_API_DB_SCHEMA_NAME", dbSchemaName)
	os.Setenv("ELASTIC_POST_INDEX_NAME", elasticsearchIndexerName)
	os.Setenv("ELASTIC_AUTO_CREATE_POST_INDEX", "true")
	defer func() {
		os.Unsetenv("APP_API_PORT")
		os.Unsetenv("APP_API_KAFKA_TOPIC")
		os.Unsetenv("APP_API_DB_SCHEMA_NAME")
		os.Unsetenv("ELASTIC_POST_INDEX_NAME")
	}()

	// Start the application in a goroutine
	errChan := make(chan error, 2)
	go func() {
		err := api_app.ListenAndServe(ctx, "") //TODO
		if err != nil {
			errChan <- err
			return
		}
	}()
	go func() {
		if err := testutils.WaitForPort(port, 15*time.Second); err != nil {
			errChan <- err
		}
		close(errChan) // close the errChan if port is accessible
	}()

	select {
	case <-ctx.Done():
		return appFuncResponse, fmt.Errorf("context cancelled. cancelled starting of startApiApp")
	case appErr := <-errChan:
		if appErr != nil {
			return appFuncResponse, appErr
		}
		appFuncResponse = testutils.AppSetupFuncResponse{
			Addr: fmt.Sprintf("http://localhost:%s", port),
		}
		return appFuncResponse, nil
	}
}

func startSearchIndexerApp(ctx context.Context) (testutils.AppSetupFuncResponse, error) {
	var appFuncResponse testutils.AppSetupFuncResponse
	if err := ctx.Err(); err != nil {
		return appFuncResponse, err
	}
	os.Setenv("APP_SEARCH_INDEXER_KAFKA_TOPIC", kafkaTopic)
	os.Setenv("APP_SEARCH_INDEXER_KAFKA_CONSUMER_GROUP_ID", kafkaConsumerGroupId)
	os.Setenv("ELASTIC_POST_INDEX_NAME", elasticsearchIndexerName)
	os.Setenv("ELASTIC_AUTO_CREATE_POST_INDEX", "true")
	defer func() {
		os.Unsetenv("APP_SEARCH_INDEXER_KAFKA_TOPIC")
		os.Unsetenv("APP_SEARCH_INDEXER_KAFKA_CONSUMER_GROUP_ID")
		os.Unsetenv("ELASTIC_POST_INDEX_NAME")
	}()

	errChan := make(chan error, 2)
	go func() {
		err := search_indexer_app.ListenAndServe(ctx, "") //TODO
		if err != nil {
			errChan <- err
			return
		}
	}()
	go func() {
		broker := os.Getenv("KAFKA_HOST")
		if broker == "" {
			errChan <- fmt.Errorf("kafka broker not available")
		}
		if err := testutils.WaitForConsumerGroup(ctx, broker, kafkaConsumerGroupId, 30*time.Second); err != nil {
			errChan <- err
		}
		close(errChan) // close the errChan if port is accessible
	}()

	select {
	case <-ctx.Done():
		return appFuncResponse, fmt.Errorf("context cancelled. cancelled starting of startSearchIndexerApp")
	case appErr := <-errChan:
		if appErr != nil {
			return appFuncResponse, appErr
		}
		return appFuncResponse, nil
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

func sendAndGetResponseBody[T any](serverAddr string, client *http.Client, testReq testutils.HttpTestRequest) (T, error) {
	var respBody T
	resp, err := testutils.SendRequest(serverAddr, client, testReq)
	if err != nil {
		return respBody, fmt.Errorf("failed to send create test user: %w", err)
	}
	defer resp.Body.Close()

	respBody, err = getResponseBody[T](resp)
	if err != nil {
		return respBody, err
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

func comparePostSearchResponse(expected, actual store.PostSearchResp) error {
	isResultsValid := func(expected, actual []store.IndexedPost) bool {
		if len(expected) != len(actual) {
			return false
		}
		for i, v := range expected {
			actualPost := actual[i]
			if actualPost.ID != v.ID {
				return false
			}
		}
		return true
	}
	if expected.Total != actual.Total || !isResultsValid(expected.Results, actual.Results) {
		return fmt.Errorf("Expected response body:%v,instead received body:%v", expected, actual)
	}
	return nil
}
