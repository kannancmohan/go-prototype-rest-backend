package app_common

import (
	esv8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/kannancmohan/go-prototype-rest-backend/internal/api/common"
)

type ElasticSearchConfig struct {
	Addr string
}

func (e *ElasticSearchConfig) NewElasticSearch() (*esv8.Client, error) {
	es, err := esv8.NewClient(esv8.Config{
		Addresses: []string{e.Addr},
		//RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries: 5,
	})

	if err != nil {
		return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "elasticsearch.Open")
	}

	res, err := es.Info()
	if err != nil {
		return nil, common.WrapErrorf(err, common.ErrorCodeUnknown, "es.Info")
	}

	defer func() {
		err = res.Body.Close()
	}()

	return es, nil
}
