/*
Copyright 2025 KubeAI Autoscaler Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockClient(t *testing.T) {
	tests := []struct {
		name     string
		mock     *MockClient
		testFunc func(ctx context.Context, client *MockClient) error
	}{
		{
			name: "GetLatencyP99 returns configured value",
			mock: &MockClient{
				LatencyP99Value: 0.5,
			},
			testFunc: func(ctx context.Context, client *MockClient) error {
				val, err := client.GetLatencyP99(ctx, "")
				if err != nil {
					return err
				}
				if val != 0.5 {
					return errors.New("unexpected value")
				}
				return nil
			},
		},
		{
			name: "GetLatencyP95 returns configured value",
			mock: &MockClient{
				LatencyP95Value: 0.3,
			},
			testFunc: func(ctx context.Context, client *MockClient) error {
				val, err := client.GetLatencyP95(ctx, "")
				if err != nil {
					return err
				}
				if val != 0.3 {
					return errors.New("unexpected value")
				}
				return nil
			},
		},
		{
			name: "GetGPUUtilization returns configured value",
			mock: &MockClient{
				GPUUtilizationValue: 75.5,
			},
			testFunc: func(ctx context.Context, client *MockClient) error {
				val, err := client.GetGPUUtilization(ctx, "")
				if err != nil {
					return err
				}
				if val != 75.5 {
					return errors.New("unexpected value")
				}
				return nil
			},
		},
		{
			name: "GetQueueDepth returns configured value",
			mock: &MockClient{
				QueueDepthValue: 100,
			},
			testFunc: func(ctx context.Context, client *MockClient) error {
				val, err := client.GetQueueDepth(ctx, "")
				if err != nil {
					return err
				}
				if val != 100 {
					return errors.New("unexpected value")
				}
				return nil
			},
		},
		{
			name: "returns configured error",
			mock: &MockClient{
				Error: errors.New("test error"),
			},
			testFunc: func(ctx context.Context, client *MockClient) error {
				_, err := client.GetLatencyP99(ctx, "")
				if err == nil {
					return errors.New("expected error")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.testFunc(ctx, tt.mock)
			assert.NoError(t, err)
		})
	}
}

func TestDefaultQueries(t *testing.T) {
	// Test that default queries are used when empty string is passed
	mock := &MockClient{
		LatencyP99Value:     0.5,
		LatencyP95Value:     0.3,
		GPUUtilizationValue: 75.0,
		QueueDepthValue:     50,
	}

	ctx := context.Background()

	// These should work with empty queries
	_, err := mock.GetLatencyP99(ctx, "")
	assert.NoError(t, err)

	_, err = mock.GetLatencyP95(ctx, "")
	assert.NoError(t, err)

	_, err = mock.GetGPUUtilization(ctx, "")
	assert.NoError(t, err)

	_, err = mock.GetQueueDepth(ctx, "")
	assert.NoError(t, err)
}

// Helper functions for generating Prometheus API responses

func makeVectorResponse(value float64) string {
	return fmt.Sprintf(`{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [{"metric": {}, "value": [1234567890.123, "%v"]}]
		}
	}`, value)
}

func makeVectorResponseMultiple(values ...float64) string {
	results := ""
	for i, v := range values {
		if i > 0 {
			results += ", "
		}
		results += fmt.Sprintf(`{"metric": {}, "value": [1234567890.123, "%v"]}`, v)
	}
	return fmt.Sprintf(`{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [%s]
		}
	}`, results)
}

func makeScalarResponse(value float64) string {
	return fmt.Sprintf(`{
		"status": "success",
		"data": {
			"resultType": "scalar",
			"result": [1234567890.123, "%v"]
		}
	}`, value)
}

func makeEmptyVectorResponse() string {
	return `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": []
		}
	}`
}

func makeErrorResponse(errorType, message string) string {
	return fmt.Sprintf(`{
		"status": "error",
		"errorType": "%s",
		"error": "%s"
	}`, errorType, message)
}

func makeMatrixResponse() string {
	return `{
		"status": "success",
		"data": {
			"resultType": "matrix",
			"result": [{"metric": {}, "values": [[1234567890.123, "123.45"]]}]
		}
	}`
}

const floatDelta = 1e-6

// TestPrometheusClient_ImplementsInterface verifies PrometheusClient implements Client interface
func TestPrometheusClient_ImplementsInterface(t *testing.T) {
	var _ Client = (*PrometheusClient)(nil)
}

func TestNewPrometheusClient(t *testing.T) {
	client, err := NewPrometheusClient("http://localhost:9090")
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.api)
}

func TestPrometheusClient_Query_Success(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectedValue  float64
	}{
		{
			name:          "vector response",
			response:      makeVectorResponse(123.45),
			expectedValue: 123.45,
		},
		{
			name:          "scalar response",
			response:      makeScalarResponse(67.89),
			expectedValue: 67.89,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client, err := NewPrometheusClient(server.URL)
			require.NoError(t, err)

			value, err := client.Query(context.Background(), "test_query")
			require.NoError(t, err)
			assert.InDelta(t, tt.expectedValue, value, floatDelta)
		})
	}
}

func TestPrometheusClient_GetLatencyP99(t *testing.T) {
	var capturedQuery string
	expectedValue := 0.150

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse form to get query - Prometheus client sends as form data
		_ = r.ParseForm()
		capturedQuery = r.FormValue("query")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(makeVectorResponse(expectedValue)))
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	value, err := client.GetLatencyP99(context.Background(), "")
	require.NoError(t, err)
	assert.InDelta(t, expectedValue, value, floatDelta)

	// Verify the default query contains expected components
	assert.Contains(t, capturedQuery, "0.99")
	assert.Contains(t, capturedQuery, "histogram_quantile")
	assert.Contains(t, capturedQuery, "inference_request_duration_seconds_bucket")
}

func TestPrometheusClient_GetLatencyP95(t *testing.T) {
	var capturedQuery string
	expectedValue := 0.120

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		capturedQuery = r.FormValue("query")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(makeVectorResponse(expectedValue)))
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	value, err := client.GetLatencyP95(context.Background(), "")
	require.NoError(t, err)
	assert.InDelta(t, expectedValue, value, floatDelta)

	// Verify the default query contains expected components
	assert.Contains(t, capturedQuery, "0.95")
	assert.Contains(t, capturedQuery, "histogram_quantile")
	assert.Contains(t, capturedQuery, "inference_request_duration_seconds_bucket")
}

func TestPrometheusClient_GetGPUUtilization(t *testing.T) {
	var capturedQuery string
	expectedValue := 75.5

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		capturedQuery = r.FormValue("query")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(makeVectorResponse(expectedValue)))
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	value, err := client.GetGPUUtilization(context.Background(), "")
	require.NoError(t, err)
	assert.InDelta(t, expectedValue, value, floatDelta)

	// Verify the default query contains expected components
	assert.Contains(t, capturedQuery, "DCGM_FI_DEV_GPU_UTIL")
	assert.Contains(t, capturedQuery, "avg")
}

func TestPrometheusClient_GetQueueDepth(t *testing.T) {
	var capturedQuery string
	expectedValue := int64(42)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		capturedQuery = r.FormValue("query")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(makeVectorResponse(float64(expectedValue))))
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	value, err := client.GetQueueDepth(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, expectedValue, value)

	// Verify the default query contains expected components
	assert.Contains(t, capturedQuery, "inference_request_queue_depth")
	assert.Contains(t, capturedQuery, "sum")
}

func TestPrometheusClient_Query_Errors(t *testing.T) {
	tests := []struct {
		name               string
		setupServer        func() *httptest.Server
		closeBeforeQuery   bool
		expectedErrContains string
	}{
		{
			name: "connection failure",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			closeBeforeQuery:   true,
		},
		{
			name: "HTTP 500 error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
		},
		{
			name: "HTTP 400 error with error JSON",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(makeErrorResponse("bad_data", "invalid query")))
				}))
			},
		},
		{
			name: "Prometheus error status",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(makeErrorResponse("execution", "query execution error")))
				}))
			},
		},
		{
			name: "empty result",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(makeEmptyVectorResponse()))
				}))
			},
			expectedErrContains: "no data returned from query:",
		},
		{
			name: "invalid JSON",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{not-json`))
				}))
			},
		},
		{
			name: "unexpected result type (matrix)",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(makeMatrixResponse()))
				}))
			},
			expectedErrContains: "unexpected result type:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			serverURL := server.URL

			if tt.closeBeforeQuery {
				server.Close()
			} else {
				defer server.Close()
			}

			client, err := NewPrometheusClient(serverURL)
			require.NoError(t, err)

			_, err = client.Query(context.Background(), "test_query")
			require.Error(t, err)
			if tt.expectedErrContains != "" {
				assert.Contains(t, err.Error(), tt.expectedErrContains)
			}
		})
	}
}

func TestPrometheusClient_GetQueueDepth_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(makeEmptyVectorResponse()))
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	_, err = client.GetQueueDepth(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no data returned from query:")
}

func TestPrometheusClient_ContextCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, qErr := client.Query(ctx, "test_query")
		errCh <- qErr
	}()

	<-started
	cancel()

	err = <-errCh
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded),
		"expected context cancellation error, got %v", err)
}

func TestPrometheusClient_Query_MultipleSamples(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(makeVectorResponseMultiple(11.11, 22.22)))
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	value, err := client.Query(context.Background(), "test_query")
	require.NoError(t, err)
	assert.InDelta(t, 11.11, value, floatDelta)
}

func TestPrometheusClient_CustomQuery(t *testing.T) {
	var capturedQuery string
	customQuery := "custom_metric{label=\"value\"}"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		capturedQuery = r.FormValue("query")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(makeVectorResponse(99.9)))
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	// Test GetLatencyP99 with custom query
	capturedQuery = ""
	value, err := client.GetLatencyP99(context.Background(), customQuery)
	require.NoError(t, err)
	assert.InDelta(t, 99.9, value, floatDelta)
	assert.Equal(t, customQuery, capturedQuery)

	// Test GetLatencyP95 with custom query
	capturedQuery = ""
	value, err = client.GetLatencyP95(context.Background(), customQuery)
	require.NoError(t, err)
	assert.InDelta(t, 99.9, value, floatDelta)
	assert.Equal(t, customQuery, capturedQuery)

	// Test GetGPUUtilization with custom query
	capturedQuery = ""
	value, err = client.GetGPUUtilization(context.Background(), customQuery)
	require.NoError(t, err)
	assert.InDelta(t, 99.9, value, floatDelta)
	assert.Equal(t, customQuery, capturedQuery)

	// Test GetQueueDepth with custom query
	capturedQuery = ""
	queueDepth, err := client.GetQueueDepth(context.Background(), customQuery)
	require.NoError(t, err)
	assert.Equal(t, int64(99), queueDepth)
	assert.Equal(t, customQuery, capturedQuery)
}

func TestPrometheusClient_Query_NonNumericSample(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [{"metric": {}, "value": [1234567890.123, "NaN"]}]
			}
		}`))
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	value, err := client.Query(context.Background(), "test_query")
	require.NoError(t, err)
	assert.True(t, math.IsNaN(value), "expected NaN value, got %v", value)
}
