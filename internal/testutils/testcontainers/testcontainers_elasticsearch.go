package testcontainers_testutils

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	esv8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/elasticsearch"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testElasticsearchContainer struct {
}

func NewTestElasticsearchContainer() *testElasticsearchContainer {
	return &testElasticsearchContainer{}
}

func (e *testElasticsearchContainer) CreateElasticsearchTestContainer(containerName string, indexMappings map[string]interface{}) (*elasticsearch.ElasticsearchContainer, func(ctx context.Context) error, error) {
	if containerName == "" {
		containerName = fmt.Sprintf("es-instance-%s", uuid.New().String())
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ctr, err := elasticsearch.Run(
		timeoutCtx,
		"elasticsearch:8.16.2",
		testcontainers.WithEnv(
			map[string]string{
				"xpack.security.enabled":     "false",
				"xpack.monitoring.exporters": "{}",
				"xpack.profiling.enabled":    "false",
				"ES_JAVA_OPTS":               "-Xms512m -Xmx512m",
			},
		),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/_cluster/health").WithStartupTimeout(1*time.Minute),
		),
		testcontainers.CustomizeRequest(additionalCustomizeRequest(containerName, indexMappings)),
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

func (e *testElasticsearchContainer) CreateElasticsearchInstance(container *elasticsearch.ElasticsearchContainer) (*esv8.Client, error) {
	addr := container.Settings.Address

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

func additionalCustomizeRequest(containerName string, indexMappings map[string]interface{}) testcontainers.GenericContainerRequest {
	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Name: containerName,
		},
	}

	// Conditionally add LifecycleHooks if indexes are provided
	if len(indexMappings) > 0 {
		req.LifecycleHooks = []testcontainers.ContainerLifecycleHooks{
			{
				PostStarts: []testcontainers.ContainerHook{
					// 1. Copy the script into the container
					func(ctx context.Context, c testcontainers.Container) error {

						host, err := c.Host(ctx) // Get the container's host
						if err != nil {
							return fmt.Errorf("failed to get container host: %w", err)
						}
						port, err := c.MappedPort(ctx, "9200/tcp") // Get the container's port
						if err != nil {
							return fmt.Errorf("failed to get mapped port: %w", err)
						}

						scriptContent := generateIndexCreationScriptContent(fmt.Sprintf("%s:%s", host, port.Port()), indexMappings)
						err = c.CopyToContainer(ctx, scriptContent, "/usr/share/elasticsearch/init-elasticsearch.sh", 0o755)
						if err != nil {
							return fmt.Errorf("failed to copy script to container: %w", err)
						}

						// Execute the script inside the container
						_, _, err = c.Exec(ctx, []string{"/bin/bash", "/usr/share/elasticsearch/init-elasticsearch.sh"})
						if err != nil {
							return fmt.Errorf("failed to execute script: %w", err)
						}

						return nil
					},
				},
			},
		}
	}
	return req
}

// generateScriptContent generates a script to create Elasticsearch indexes with optional mappings
func generateIndexCreationScriptContent(host string, indexMappings map[string]interface{}) []byte {
	script := `#!/bin/bash

# Wait for Elasticsearch to start
until curl -s http://%s; do
  echo "Waiting for Elasticsearch to start..."
  sleep 2
done

# Create indexes with optional mappings
`
	script = fmt.Sprintf(script, host)
	for indexName, mappings := range indexMappings {
		script += fmt.Sprintf(`
curl -X PUT "http://%s/%s" -H 'Content-Type: application/json' -d'
{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  }`, host, indexName)
		if len(mappings.(map[string]interface{})) > 0 { // Include mappings only if they are not empty
			mappingsJSON, err := json.Marshal(mappings)
			if err != nil {
				panic(fmt.Errorf("failed to marshal mappings for index %s: %w", indexName, err))
			}
			script += fmt.Sprintf(`,
  "mappings": %s`, string(mappingsJSON))
		}
		script += `
}'
`
	}

	script += `
echo "Indexes created successfully!"
`

	return []byte(script)
}
