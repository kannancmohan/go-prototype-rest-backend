package app_common

import (
	esv8 "github.com/elastic/go-elasticsearch/v8"
)

type ElasticSearchConfig struct {
	Addr string
}

func (e *ElasticSearchConfig) NewConnection() (*esv8.Client, error) {
	es, err := esv8.NewClient(esv8.Config{
		Addresses: []string{e.Addr},
		//RetryOnStatus: []int{502, 503, 504, 429},
		EnableDebugLogger: true,
		MaxRetries:        5,
	})

	if err != nil {
		return nil, err
	}

	res, err := es.Info()
	if err != nil {
		return nil, err
	}

	defer func() {
		err = res.Body.Close()
	}()

	return es, nil
}
