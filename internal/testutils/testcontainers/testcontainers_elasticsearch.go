package testcontainers_testutils

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	esv8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/elasticsearch"
)

type testElasticsearchContainer struct {
}

func NewTestElasticsearchContainer() *testElasticsearchContainer {
	return &testElasticsearchContainer{}
}

func (e *testElasticsearchContainer) CreateElasticsearchTestContainer(containerName string) (*elasticsearch.ElasticsearchContainer, func(ctx context.Context) error, error) {
	if containerName == "" {
		containerName = fmt.Sprintf("es-instance-%s", uuid.New().String())
	}
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ctr, err := elasticsearch.Run(
		timeoutCtx,
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
