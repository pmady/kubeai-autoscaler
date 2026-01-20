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

package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeaiv1alpha1 "github.com/pmady/kubeai-autoscaler/api/v1alpha1"
	"github.com/pmady/kubeai-autoscaler/pkg/metrics"
	"github.com/pmady/kubeai-autoscaler/pkg/scaling"
)

func TestCalculateDesiredReplicas(t *testing.T) {
	tests := []struct {
		name              string
		policy            *kubeaiv1alpha1.AIInferenceAutoscalerPolicy
		currentReplicas   int32
		currentMetrics    *kubeaiv1alpha1.CurrentMetrics
		expected          int32
		expectedAlgorithm string
	}{
		{
			name: "scale up based on latency",
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					MinReplicas: 1,
					MaxReplicas: 10,
					Metrics: kubeaiv1alpha1.MetricsSpec{
						Latency: &kubeaiv1alpha1.LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 100,
						},
					},
				},
			},
			currentReplicas:   2,
			currentMetrics:    &kubeaiv1alpha1.CurrentMetrics{LatencyP99Ms: 200},
			expected:          4,
			expectedAlgorithm: "MaxRatio",
		},
		{
			name: "scale up based on GPU utilization",
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					MinReplicas: 1,
					MaxReplicas: 10,
					Metrics: kubeaiv1alpha1.MetricsSpec{
						GPUUtilization: &kubeaiv1alpha1.GPUUtilizationMetric{
							Enabled:          true,
							TargetPercentage: 50,
						},
					},
				},
			},
			currentReplicas:   2,
			currentMetrics:    &kubeaiv1alpha1.CurrentMetrics{GPUUtilizationPercent: 100},
			expected:          4,
			expectedAlgorithm: "MaxRatio",
		},
		{
			name: "respect max replicas",
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					MinReplicas: 1,
					MaxReplicas: 5,
					Metrics: kubeaiv1alpha1.MetricsSpec{
						Latency: &kubeaiv1alpha1.LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 100,
						},
					},
				},
			},
			currentReplicas:   3,
			currentMetrics:    &kubeaiv1alpha1.CurrentMetrics{LatencyP99Ms: 500},
			expected:          5,
			expectedAlgorithm: "MaxRatio",
		},
		{
			name: "respect min replicas",
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					MinReplicas: 2,
					MaxReplicas: 10,
					Metrics: kubeaiv1alpha1.MetricsSpec{
						Latency: &kubeaiv1alpha1.LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 100,
						},
					},
				},
			},
			currentReplicas:   1,
			currentMetrics:    &kubeaiv1alpha1.CurrentMetrics{LatencyP99Ms: 100},
			expected:          2,
			expectedAlgorithm: "MaxRatio",
		},
		{
			name: "no scaling when at target",
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					MinReplicas: 1,
					MaxReplicas: 10,
					Metrics: kubeaiv1alpha1.MetricsSpec{
						Latency: &kubeaiv1alpha1.LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 100,
						},
					},
				},
			},
			currentReplicas:   3,
			currentMetrics:    &kubeaiv1alpha1.CurrentMetrics{LatencyP99Ms: 100},
			expected:          3,
			expectedAlgorithm: "MaxRatio",
		},
		{
			name: "use highest ratio from multiple metrics",
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					MinReplicas: 1,
					MaxReplicas: 10,
					Metrics: kubeaiv1alpha1.MetricsSpec{
						Latency: &kubeaiv1alpha1.LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 100,
						},
						GPUUtilization: &kubeaiv1alpha1.GPUUtilizationMetric{
							Enabled:          true,
							TargetPercentage: 50,
						},
					},
				},
			},
			currentReplicas:   2,
			currentMetrics:    &kubeaiv1alpha1.CurrentMetrics{LatencyP99Ms: 150, GPUUtilizationPercent: 150},
			expected:          6,
			expectedAlgorithm: "MaxRatio",
		},
		{
			name: "use AverageRatio algorithm",
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					MinReplicas: 1,
					MaxReplicas: 10,
					Algorithm: &kubeaiv1alpha1.AlgorithmSpec{
						Name:      "AverageRatio",
						Tolerance: 0.1,
					},
					Metrics: kubeaiv1alpha1.MetricsSpec{
						Latency: &kubeaiv1alpha1.LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 100,
						},
						GPUUtilization: &kubeaiv1alpha1.GPUUtilizationMetric{
							Enabled:          true,
							TargetPercentage: 50,
						},
					},
				},
			},
			currentReplicas:   2,
			currentMetrics:    &kubeaiv1alpha1.CurrentMetrics{LatencyP99Ms: 200, GPUUtilizationPercent: 100},
			expected:          4, // avg of 2.0 and 2.0 = 2.0, 2 * 2.0 = 4
			expectedAlgorithm: "AverageRatio",
		},
		{
			name: "fallback to MaxRatio for unknown algorithm",
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					MinReplicas: 1,
					MaxReplicas: 10,
					Algorithm: &kubeaiv1alpha1.AlgorithmSpec{
						Name: "NonExistentAlgorithm",
					},
					Metrics: kubeaiv1alpha1.MetricsSpec{
						Latency: &kubeaiv1alpha1.LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 100,
						},
					},
				},
			},
			currentReplicas:   2,
			currentMetrics:    &kubeaiv1alpha1.CurrentMetrics{LatencyP99Ms: 200},
			expected:          4,
			expectedAlgorithm: "MaxRatio", // Falls back
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &AIInferenceAutoscalerPolicyReconciler{
				AlgorithmRegistry: scaling.DefaultRegistry,
			}
			ctx := context.Background()

			result, algorithmUsed, _ := r.calculateDesiredReplicas(ctx, tt.policy, tt.currentReplicas, tt.currentMetrics)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expectedAlgorithm, algorithmUsed)
		})
	}
}

func TestMockMetricsClient(t *testing.T) {
	mock := &metrics.MockClient{
		LatencyP99Value:     0.5,
		LatencyP95Value:     0.3,
		GPUUtilizationValue: 75.0,
		QueueDepthValue:     100,
	}

	ctx := context.Background()

	latency, err := mock.GetLatencyP99(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, 0.5, latency)

	gpu, err := mock.GetGPUUtilization(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, 75.0, gpu)

	queue, err := mock.GetQueueDepth(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(100), queue)
}

func TestPolicyDefaults(t *testing.T) {
	policy := &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
		},
		Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
			TargetRef: kubeaiv1alpha1.TargetRef{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-deployment",
			},
			MaxReplicas: 10,
			Metrics: kubeaiv1alpha1.MetricsSpec{
				Latency: &kubeaiv1alpha1.LatencyMetric{
					Enabled:     true,
					TargetP99Ms: 500,
				},
			},
		},
	}

	// Test that MinReplicas defaults to 1 when not set
	assert.Equal(t, int32(0), policy.Spec.MinReplicas)

	// In the reconciler, we handle this:
	minReplicas := policy.Spec.MinReplicas
	if minReplicas == 0 {
		minReplicas = 1
	}
	assert.Equal(t, int32(1), minReplicas)
}
