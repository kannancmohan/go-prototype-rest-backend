package testutils

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	esv8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/elasticsearch"
)

var (
	esContainerInstances sync.Map
	esMutex              sync.Mutex
)

type ESContainerInfo struct {
	Container testcontainers.Container
	ESClient  *esv8.Client
}

type ESCleanupFunc func(ctx context.Context) error

func StartElasticsearchTestContainer(containerName string) (*esv8.Client, ESCleanupFunc, error) {
	if containerName == "" {
		containerName = fmt.Sprintf("es-instance-%s", uuid.New().String())
	}

	esMutex.Lock()
	defer esMutex.Unlock()

	ctx := context.Background()

	// Check if a container for this containerName already exists
	if instance, ok := esContainerInstances.Load(containerName); ok {
		info := instance.(*ESContainerInfo)
		return info.ESClient, func(ctx context.Context) error { return nil }, nil // No-op cleanup for reused container
	}

	container, err := createElasticsearchTestContainer(ctx, containerName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start Elasticsearch container: %w", err)
	}

	addr := container.Settings.Address

	client, err := createElasticsearchInstance(addr)
	if err != nil {
		container.Terminate(ctx) // Ensure cleanup
		return nil, nil, fmt.Errorf("failed to create Elasticsearch instance: %w", err)
	}

	esContainerInstances.Store(containerName, &ESContainerInfo{
		Container: container,
		ESClient:  client,
	})

	cleanupFunc := func(ctx context.Context) error {
		esMutex.Lock()
		defer esMutex.Unlock()

		if instance, ok := esContainerInstances.Load(containerName); ok {
			esContainerInstances.Delete(containerName)

			info := instance.(*ESContainerInfo)
			if info.Container != nil {
				if err := info.Container.Terminate(ctx); err != nil {
					return fmt.Errorf("failed to terminate Elasticsearch container: %w", err)
				}
			}
		}
		return nil
	}

	return client, cleanupFunc, nil

}

func createElasticsearchTestContainer(ctx context.Context, containerName string) (*elasticsearch.ElasticsearchContainer, error) {

	container, err := elasticsearch.Run(
		ctx,
		"docker.elastic.co/elasticsearch/elasticsearch:8.16.2",
		testcontainers.WithEnv(
			map[string]string{
				"xpack.security.enabled": "false",
				"ES_JAVA_OPTS":           "-Xms512m -Xmx512m",
			},
		),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Name: containerName,
			},
		}),
	)

	if err != nil {
		return nil, err
	}
	return container, nil
}

func createElasticsearchInstance(addr string) (*esv8.Client, error) {
	client, err := esv8.NewClient(esv8.Config{
		Addresses:         []string{addr},
		EnableDebugLogger: true,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	})

	if err != nil {
		return nil, err
	}

	resp, err := client.Info()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return client, nil
}
