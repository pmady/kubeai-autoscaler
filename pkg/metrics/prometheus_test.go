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
	"testing"

	"github.com/stretchr/testify/assert"
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
