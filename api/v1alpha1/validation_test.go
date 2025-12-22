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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidation(t *testing.T) {
	tests := []struct {
		name        string
		policy      *AIInferenceAutoscalerPolicy
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid policy",
			policy: &AIInferenceAutoscalerPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-policy",
					Namespace: "default",
				},
				Spec: AIInferenceAutoscalerPolicySpec{
					TargetRef: TargetRef{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "test-deployment",
					},
					MinReplicas: 1,
					MaxReplicas: 10,
					Metrics: MetricsSpec{
						Latency: &LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 500,
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing target name",
			policy: &AIInferenceAutoscalerPolicy{
				Spec: AIInferenceAutoscalerPolicySpec{
					TargetRef: TargetRef{
						Kind: "Deployment",
						Name: "",
					},
					MaxReplicas: 10,
					Metrics: MetricsSpec{
						Latency: &LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 500,
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "targetRef.name is required",
		},
		{
			name: "invalid target kind",
			policy: &AIInferenceAutoscalerPolicy{
				Spec: AIInferenceAutoscalerPolicySpec{
					TargetRef: TargetRef{
						Kind: "DaemonSet",
						Name: "test",
					},
					MaxReplicas: 10,
					Metrics: MetricsSpec{
						Latency: &LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 500,
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "targetRef.kind must be Deployment or StatefulSet",
		},
		{
			name: "maxReplicas zero",
			policy: &AIInferenceAutoscalerPolicy{
				Spec: AIInferenceAutoscalerPolicySpec{
					TargetRef: TargetRef{
						Kind: "Deployment",
						Name: "test",
					},
					MaxReplicas: 0,
					Metrics: MetricsSpec{
						Latency: &LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 500,
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "maxReplicas must be greater than 0",
		},
		{
			name: "minReplicas greater than maxReplicas",
			policy: &AIInferenceAutoscalerPolicy{
				Spec: AIInferenceAutoscalerPolicySpec{
					TargetRef: TargetRef{
						Kind: "Deployment",
						Name: "test",
					},
					MinReplicas: 10,
					MaxReplicas: 5,
					Metrics: MetricsSpec{
						Latency: &LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 500,
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "minReplicas cannot be greater than maxReplicas",
		},
		{
			name: "no metrics enabled",
			policy: &AIInferenceAutoscalerPolicy{
				Spec: AIInferenceAutoscalerPolicySpec{
					TargetRef: TargetRef{
						Kind: "Deployment",
						Name: "test",
					},
					MaxReplicas: 10,
					Metrics:     MetricsSpec{},
				},
			},
			expectError: true,
			errorMsg:    "at least one metric must be enabled",
		},
		{
			name: "latency enabled but no target",
			policy: &AIInferenceAutoscalerPolicy{
				Spec: AIInferenceAutoscalerPolicySpec{
					TargetRef: TargetRef{
						Kind: "Deployment",
						Name: "test",
					},
					MaxReplicas: 10,
					Metrics: MetricsSpec{
						Latency: &LatencyMetric{
							Enabled: true,
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "latency metric enabled but no target specified",
		},
		{
			name: "GPU utilization out of range",
			policy: &AIInferenceAutoscalerPolicy{
				Spec: AIInferenceAutoscalerPolicySpec{
					TargetRef: TargetRef{
						Kind: "Deployment",
						Name: "test",
					},
					MaxReplicas: 10,
					Metrics: MetricsSpec{
						GPUUtilization: &GPUUtilizationMetric{
							Enabled:          true,
							TargetPercentage: 150,
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "gpuUtilization.targetPercentage must be between 1 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	policy := &AIInferenceAutoscalerPolicy{
		Spec: AIInferenceAutoscalerPolicySpec{
			TargetRef: TargetRef{
				Kind: "Deployment",
				Name: "test",
			},
			MaxReplicas: 10,
		},
	}

	policy.SetDefaults()

	assert.Equal(t, int32(1), policy.Spec.MinReplicas)
	assert.Equal(t, int32(300), policy.Spec.CooldownPeriod)
	assert.Equal(t, "apps/v1", policy.Spec.TargetRef.APIVersion)
}
