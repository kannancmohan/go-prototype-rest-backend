package testutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/template"
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
	Error  interface{} `json:"error" yaml:"error"`
}

func LoadTestCases(file string, placeholders map[string]interface{}) ([]HttpTestCase, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	jsonStr := string(data)
	if len(placeholders) > 0 {
		jsonStr, err = applyTemplate(jsonStr, placeholders)
		if err != nil {
			return nil, err
		}
	}

	var testCases []HttpTestCase
	if err := json.Unmarshal([]byte(jsonStr), &testCases); err != nil {
		return nil, err
	}

	return testCases, nil
}

func applyTemplate(jsonStr string, placeholders map[string]interface{}) (string, error) {
	funcMap := template.FuncMap{
		// Define a custom 'json' function to marshal values into raw JSON
		"json": func(v interface{}) (string, error) {
			bytes, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(bytes), nil
		},
	}

	tmpl, err := template.New("json").Funcs(funcMap).Parse(jsonStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, placeholders); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func SendRequest(serverAddr string, client *http.Client, testReq HttpTestRequest) (*http.Response, error) {
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
