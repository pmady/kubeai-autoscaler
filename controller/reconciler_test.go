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

package main

import (
	"context"
	"testing"
	"time"

	kubeaiv1alpha1 "github.com/pmady/kubeai-autoscaler/api/v1alpha1"
	"github.com/pmady/kubeai-autoscaler/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCalculateDesiredReplicasReconciler(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kubeaiv1alpha1.AddToScheme(scheme)

	tests := []struct {
		name            string
		currentReplicas int32
		policy          *kubeaiv1alpha1.AIInferenceAutoscalerPolicy
		currentMetrics  *kubeaiv1alpha1.CurrentMetrics
		expectedResult  int32
	}{
		{
			name:            "scale up based on high latency",
			currentReplicas: 2,
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
			currentMetrics: &kubeaiv1alpha1.CurrentMetrics{
				LatencyP99Ms: 200, // 2x target
			},
			expectedResult: 4, // 2 * 2 = 4
		},
		{
			name:            "scale up based on high GPU utilization",
			currentReplicas: 3,
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					MinReplicas: 1,
					MaxReplicas: 10,
					Metrics: kubeaiv1alpha1.MetricsSpec{
						GPUUtilization: &kubeaiv1alpha1.GPUUtilizationMetric{
							Enabled:          true,
							TargetPercentage: 80,
						},
					},
				},
			},
			currentMetrics: &kubeaiv1alpha1.CurrentMetrics{
				GPUUtilizationPercent: 100, // 125% of target
			},
			expectedResult: 4, // ceil(3 * 1.25) = 4
		},
		{
			name:            "respect max replicas limit",
			currentReplicas: 8,
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
			currentMetrics: &kubeaiv1alpha1.CurrentMetrics{
				LatencyP99Ms: 500, // 5x target would be 40 replicas
			},
			expectedResult: 10, // clamped to max
		},
		{
			name:            "respect min replicas limit",
			currentReplicas: 2,
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					MinReplicas: 3,
					MaxReplicas: 10,
					Metrics: kubeaiv1alpha1.MetricsSpec{
						Latency: &kubeaiv1alpha1.LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 100,
						},
					},
				},
			},
			currentMetrics: &kubeaiv1alpha1.CurrentMetrics{
				LatencyP99Ms: 100, // at target, ratio = 1
			},
			expectedResult: 3, // clamped to min since current (2) < min (3)
		},
		{
			name:            "no scaling when metrics are nil",
			currentReplicas: 5,
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
			currentMetrics: nil,
			expectedResult: 5, // no change
		},
		{
			name:            "use highest ratio from multiple metrics",
			currentReplicas: 2,
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
							TargetPercentage: 80,
						},
					},
				},
			},
			currentMetrics: &kubeaiv1alpha1.CurrentMetrics{
				LatencyP99Ms:          150, // 1.5x
				GPUUtilizationPercent: 160, // 2x (higher)
			},
			expectedResult: 4, // uses GPU ratio: ceil(2 * 2) = 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			reconciler := NewAIInferenceAutoscalerPolicyReconciler(client, scheme, nil)

			result := reconciler.calculateDesiredReplicas(
				context.Background(),
				tt.policy,
				tt.currentReplicas,
				tt.currentMetrics,
			)

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestCooldownPeriod(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kubeaiv1alpha1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := NewAIInferenceAutoscalerPolicyReconciler(client, scheme, nil)

	policyKey := "test-namespace/test-policy"

	// First scale should be allowed
	assert.True(t, reconciler.canScale(policyKey, 300))

	// Set last scale time
	reconciler.setLastScaleTime(policyKey, time.Now())

	// Immediate scale should be blocked
	assert.False(t, reconciler.canScale(policyKey, 300))

	// Set last scale time to past
	reconciler.setLastScaleTime(policyKey, time.Now().Add(-10*time.Minute))

	// Scale should be allowed after cooldown
	assert.True(t, reconciler.canScale(policyKey, 300))
}

func TestGetCurrentReplicas(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kubeaiv1alpha1.AddToScheme(scheme)

	replicas := int32(5)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deployment).
		Build()

	reconciler := NewAIInferenceAutoscalerPolicyReconciler(client, scheme, nil)

	policy := &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "test-namespace",
		},
		Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
			TargetRef: kubeaiv1alpha1.TargetRef{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "test-deployment",
			},
		},
	}

	result, err := reconciler.getCurrentReplicas(context.Background(), policy)
	require.NoError(t, err)
	assert.Equal(t, int32(5), result)
}

func TestScaleDeployment(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kubeaiv1alpha1.AddToScheme(scheme)

	replicas := int32(2)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(deployment).
		Build()

	reconciler := NewAIInferenceAutoscalerPolicyReconciler(client, scheme, nil)

	err := reconciler.scaleDeployment(context.Background(), "test-namespace", "test-deployment", 5)
	require.NoError(t, err)

	// Verify the deployment was scaled
	updatedDeployment := &appsv1.Deployment{}
	err = client.Get(context.Background(), types.NamespacedName{
		Namespace: "test-namespace",
		Name:      "test-deployment",
	}, updatedDeployment)
	require.NoError(t, err)
	assert.Equal(t, int32(5), *updatedDeployment.Spec.Replicas)
}

func TestFetchMetrics(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = kubeaiv1alpha1.AddToScheme(scheme)

	mockClient := &metrics.MockClient{
		LatencyP99Value:     0.5,  // 500ms
		GPUUtilizationValue: 85.0, // 85%
		QueueDepthValue:     10,
	}

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := NewAIInferenceAutoscalerPolicyReconciler(client, scheme, mockClient)

	policy := &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
		Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
			Metrics: kubeaiv1alpha1.MetricsSpec{
				Latency: &kubeaiv1alpha1.LatencyMetric{
					Enabled:     true,
					TargetP99Ms: 100,
				},
				GPUUtilization: &kubeaiv1alpha1.GPUUtilizationMetric{
					Enabled:          true,
					TargetPercentage: 80,
				},
				RequestQueueDepth: &kubeaiv1alpha1.QueueDepthMetric{
					Enabled:     true,
					TargetDepth: 5,
				},
			},
		},
	}

	currentMetrics, err := reconciler.fetchMetrics(context.Background(), policy)
	require.NoError(t, err)

	assert.Equal(t, int32(500), currentMetrics.LatencyP99Ms)
	assert.Equal(t, int32(85), currentMetrics.GPUUtilizationPercent)
	assert.Equal(t, int32(10), currentMetrics.RequestQueueDepth)
}
