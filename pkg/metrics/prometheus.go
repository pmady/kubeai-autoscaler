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
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Client interface for fetching metrics
type Client interface {
	GetLatencyP99(ctx context.Context, query string) (float64, error)
	GetLatencyP95(ctx context.Context, query string) (float64, error)
	GetGPUUtilization(ctx context.Context, query string) (float64, error)
	GetQueueDepth(ctx context.Context, query string) (int64, error)
	Query(ctx context.Context, query string) (float64, error)
}

// PrometheusClient implements the Client interface using Prometheus
type PrometheusClient struct {
	api v1.API
}

// NewPrometheusClient creates a new Prometheus client
func NewPrometheusClient(address string) (*PrometheusClient, error) {
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return &PrometheusClient{
		api: v1.NewAPI(client),
	}, nil
}

// Query executes a Prometheus query and returns the result as a float64
func (c *PrometheusClient) Query(ctx context.Context, query string) (float64, error) {
	result, warnings, err := c.api.Query(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("prometheus query failed: %w", err)
	}

	if len(warnings) > 0 {
		// Log warnings but don't fail
		fmt.Printf("Prometheus query warnings: %v\n", warnings)
	}

	switch v := result.(type) {
	case model.Vector:
		if len(v) == 0 {
			return 0, fmt.Errorf("no data returned from query: %s", query)
		}
		return float64(v[0].Value), nil
	case *model.Scalar:
		return float64(v.Value), nil
	default:
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}
}

// GetLatencyP99 fetches P99 latency metric
func (c *PrometheusClient) GetLatencyP99(ctx context.Context, query string) (float64, error) {
	if query == "" {
		query = `histogram_quantile(0.99, sum(rate(inference_request_duration_seconds_bucket[5m])) by (le))`
	}
	return c.Query(ctx, query)
}

// GetLatencyP95 fetches P95 latency metric
func (c *PrometheusClient) GetLatencyP95(ctx context.Context, query string) (float64, error) {
	if query == "" {
		query = `histogram_quantile(0.95, sum(rate(inference_request_duration_seconds_bucket[5m])) by (le))`
	}
	return c.Query(ctx, query)
}

// GetGPUUtilization fetches GPU utilization metric
func (c *PrometheusClient) GetGPUUtilization(ctx context.Context, query string) (float64, error) {
	if query == "" {
		query = `avg(DCGM_FI_DEV_GPU_UTIL)`
	}
	return c.Query(ctx, query)
}

// GetQueueDepth fetches request queue depth metric
func (c *PrometheusClient) GetQueueDepth(ctx context.Context, query string) (int64, error) {
	if query == "" {
		query = `sum(inference_request_queue_depth)`
	}
	value, err := c.Query(ctx, query)
	if err != nil {
		return 0, err
	}
	return int64(value), nil
}

// MockClient is a mock implementation for testing
type MockClient struct {
	LatencyP99Value     float64
	LatencyP95Value     float64
	GPUUtilizationValue float64
	QueueDepthValue     int64
	QueryValue          float64
	Error               error
}

// Query returns the mock query value
func (m *MockClient) Query(_ context.Context, _ string) (float64, error) {
	return m.QueryValue, m.Error
}

// GetLatencyP99 returns the mock P99 latency value
func (m *MockClient) GetLatencyP99(_ context.Context, _ string) (float64, error) {
	return m.LatencyP99Value, m.Error
}

// GetLatencyP95 returns the mock P95 latency value
func (m *MockClient) GetLatencyP95(_ context.Context, _ string) (float64, error) {
	return m.LatencyP95Value, m.Error
}

// GetGPUUtilization returns the mock GPU utilization value
func (m *MockClient) GetGPUUtilization(_ context.Context, _ string) (float64, error) {
	return m.GPUUtilizationValue, m.Error
}

// GetQueueDepth returns the mock queue depth value
func (m *MockClient) GetQueueDepth(_ context.Context, _ string) (int64, error) {
	return m.QueueDepthValue, m.Error
}
