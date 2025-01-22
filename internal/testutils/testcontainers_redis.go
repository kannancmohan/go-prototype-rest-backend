package testutils

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	redistc "github.com/testcontainers/testcontainers-go/modules/redis"
)

var (
	redisContainerInstances sync.Map // Map to store containers by schema name
	redisMutex              sync.Mutex
)

type RedisContainerInfo struct {
	Container   testcontainers.Container
	RedisClient *redis.Client
}

type RedisCleanupFunc func(ctx context.Context) error

func StartRedisTestContainer(containerName string) (*redis.Client, RedisCleanupFunc, error) {
	if containerName == "" {
		containerName = fmt.Sprintf("redis-instance-%s", uuid.New().String())
	}

	redisMutex.Lock()
	defer redisMutex.Unlock()

	ctx := context.Background()

	// Check if a container for this containerName already exists
	if instance, ok := redisContainerInstances.Load(containerName); ok {
		info := instance.(*RedisContainerInfo)
		return info.RedisClient, func(ctx context.Context) error { return nil }, nil // No-op cleanup for reused container
	}

	container, err := createRedisTestContainer(ctx, containerName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start redis container: %w", err)
	}
	connStr, err := getRedisConnectionString(ctx, container)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get redis connection string: %w", err)
	}
	rClient, err := createRedisInstance(ctx, connStr)
	if err != nil {
		container.Terminate(ctx) // Ensure cleanup
		return nil, nil, fmt.Errorf("failed to initialize redis: %w", err)
	}
	redisContainerInstances.Store(containerName, &RedisContainerInfo{
		Container:   container,
		RedisClient: rClient,
	})

	cleanupFunc := func(ctx context.Context) error {
		redisMutex.Lock()
		defer redisMutex.Unlock()

		if instance, ok := redisContainerInstances.Load(containerName); ok {
			redisContainerInstances.Delete(containerName)

			info := instance.(*RedisContainerInfo)
			info.RedisClient.Close() // Close the connection

			err := info.Container.Terminate(ctx)
			if err != nil {
				return fmt.Errorf("failed to terminate redis container: %w", err)
			}
		}
		return nil
	}

	return rClient, cleanupFunc, nil
}

func createRedisTestContainer(ctx context.Context, containerName string) (*redistc.RedisContainer, error) {
	container, err := redistc.Run(ctx,
		"redis:7-alpine",
		//redistc.WithSnapshotting(10, 1),
		redistc.WithLogLevel(redistc.LogLevelVerbose),
		//testcontainers.WithLogger(testcontainers.TestLogger(t)),
		//WithTestLogConsumer(t),
		// testcontainers.WithWaitStrategy(
		// 	wait.ForAll(
		// 		wait.ForLog("* Ready to accept connections"),
		// 		wait.ForExposedPort(),
		// 		wait.ForListeningPort(nat.Port(exposedPort)),
		// 		wait.ForExec(waitReadyCmd),
		// 	),
		// ),
	)
	if err != nil {
		return nil, err
	}
	return container, nil
}

func getRedisConnectionString(ctx context.Context, container *redistc.RedisContainer) (string, error) {
	uri, err := container.ConnectionString(ctx)
	if err != nil {
		return "", err
	}
	return uri, nil
}

func createRedisInstance(ctx context.Context, connStr string) (*redis.Client, error) {
	options, err := redis.ParseURL(connStr)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(options)

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return client, nil
}
