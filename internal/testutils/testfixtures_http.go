package testutils

import (
	"encoding/json"
	"os"
)

type HttpTestCase struct {
	Name     string               `json:"name" yaml:"name"`
	Request  HttpTestRequest      `json:"request" yaml:"request"`
	Expected HttpTestExpectedResp `json:"expected" yaml:"expected"`
}

type HttpTestRequest struct {
	Method  string            `json:"method" yaml:"method"`
	Path    string            `json:"path" yaml:"path"`
	Headers map[string]string `json:"headers" yaml:"headers"`
	Body    interface{}       `json:"body" yaml:"body"`
}

type HttpTestExpectedResp struct {
	Status int         `json:"status" yaml:"status"`
	Body   interface{} `json:"body" yaml:"body"`
}

func loadTestCases(file string) ([]HttpTestCase, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var testCases []HttpTestCase
	if err := json.Unmarshal(data, &testCases); err != nil {
		return nil, err
	}

	return testCases, nil
}
