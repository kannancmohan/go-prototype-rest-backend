package testcontainers_testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	redistc "github.com/testcontainers/testcontainers-go/modules/redis"
)

type testRedisContainer struct {
}

func NewTestRedisContainer() *testRedisContainer {
	return &testRedisContainer{}
}

func (p *testRedisContainer) CreateRedisTestContainer(containerName string) (*redistc.RedisContainer, func(ctx context.Context) error, error) {
	if containerName == "" {
		containerName = fmt.Sprintf("redis-instance-%s", uuid.New().String())
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ctr, err := redistc.Run(timeoutCtx,
		"redis:7-alpine",
		//redistc.WithSnapshotting(10, 1),
		redistc.WithLogLevel(redistc.LogLevelVerbose),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Name: containerName,
			},
		}),
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

func (p *testRedisContainer) GetRedisConnectionString(container *redistc.RedisContainer) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	uri, err := container.ConnectionString(timeoutCtx)
	if err != nil {
		return "", err
	}
	return uri, nil
}

func (p *testRedisContainer) CreateRedisInstance(connStr string) (*redis.Client, error) {
	options, err := redis.ParseURL(connStr)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(options)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Ping(timeoutCtx).Err(); err != nil {
		return nil, err
	}
	return client, nil
}
