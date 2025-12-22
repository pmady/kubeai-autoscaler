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

package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeaiv1alpha1 "github.com/pmady/kubeai-autoscaler/api/v1alpha1"
)

func TestWebhookDefault(t *testing.T) {
	webhook := &AIInferenceAutoscalerPolicyWebhook{}

	policy := &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: "default",
		},
		Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
			TargetRef: kubeaiv1alpha1.TargetRef{
				Kind: "Deployment",
				Name: "test-deployment",
			},
			MaxReplicas: 10,
		},
	}

	err := webhook.Default(context.Background(), policy)
	assert.NoError(t, err)

	// Check defaults were applied
	assert.Equal(t, int32(1), policy.Spec.MinReplicas)
	assert.Equal(t, int32(300), policy.Spec.CooldownPeriod)
	assert.Equal(t, "apps/v1", policy.Spec.TargetRef.APIVersion)
}

func TestWebhookValidateCreate(t *testing.T) {
	webhook := &AIInferenceAutoscalerPolicyWebhook{}

	tests := []struct {
		name        string
		policy      *kubeaiv1alpha1.AIInferenceAutoscalerPolicy
		expectError bool
	}{
		{
			name: "valid policy",
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					TargetRef: kubeaiv1alpha1.TargetRef{
						Kind: "Deployment",
						Name: "test",
					},
					MaxReplicas: 10,
					Metrics: kubeaiv1alpha1.MetricsSpec{
						Latency: &kubeaiv1alpha1.LatencyMetric{
							Enabled:     true,
							TargetP99Ms: 500,
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid policy - no metrics",
			policy: &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
				Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
					TargetRef: kubeaiv1alpha1.TargetRef{
						Kind: "Deployment",
						Name: "test",
					},
					MaxReplicas: 10,
					Metrics:     kubeaiv1alpha1.MetricsSpec{},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := webhook.ValidateCreate(context.Background(), tt.policy)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhookValidateUpdate(t *testing.T) {
	webhook := &AIInferenceAutoscalerPolicyWebhook{}

	oldPolicy := &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
		Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
			TargetRef: kubeaiv1alpha1.TargetRef{
				Kind: "Deployment",
				Name: "original-deployment",
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

	// Test changing target name generates warning
	newPolicy := &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{
		Spec: kubeaiv1alpha1.AIInferenceAutoscalerPolicySpec{
			TargetRef: kubeaiv1alpha1.TargetRef{
				Kind: "Deployment",
				Name: "new-deployment",
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

	warnings, err := webhook.ValidateUpdate(context.Background(), oldPolicy, newPolicy)
	assert.NoError(t, err)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "targetRef.name is being changed")
}

func TestWebhookValidateDelete(t *testing.T) {
	webhook := &AIInferenceAutoscalerPolicyWebhook{}

	policy := &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{}
	warnings, err := webhook.ValidateDelete(context.Background(), policy)
	assert.NoError(t, err)
	assert.Nil(t, warnings)
}
