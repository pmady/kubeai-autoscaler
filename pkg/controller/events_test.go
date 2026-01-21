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
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeaiv1alpha1 "github.com/pmady/kubeai-autoscaler/api/v1alpha1"
)

func TestEventRecorderNilSafe(_ *testing.T) {
	// Test that EventRecorder methods don't panic with nil recorder
	recorder := NewEventRecorder(nil)

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
		},
	}

	// These should not panic
	recorder.RecordScaleUp(policy, 2, 4)
	recorder.RecordScaleDown(policy, 4, 2)
	recorder.RecordScalingFailed(policy, errors.New("test error"))
	recorder.RecordMetricsFailed(policy, errors.New("test error"))
	recorder.RecordTargetNotFound(policy, errors.New("test error"))
	recorder.RecordCooldown(policy, 60)
}
