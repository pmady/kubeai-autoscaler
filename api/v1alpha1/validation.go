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
	"fmt"
)

// Validate validates the AIInferenceAutoscalerPolicy
func (p *AIInferenceAutoscalerPolicy) Validate() error {
	if err := p.Spec.Validate(); err != nil {
		return fmt.Errorf("spec validation failed: %w", err)
	}
	return nil
}

// Validate validates the AIInferenceAutoscalerPolicySpec
func (s *AIInferenceAutoscalerPolicySpec) Validate() error {
	// Validate TargetRef
	if s.TargetRef.Name == "" {
		return fmt.Errorf("targetRef.name is required")
	}
	if s.TargetRef.Kind != "Deployment" && s.TargetRef.Kind != "StatefulSet" {
		return fmt.Errorf("targetRef.kind must be Deployment or StatefulSet")
	}

	// Validate replicas
	if s.MaxReplicas <= 0 {
		return fmt.Errorf("maxReplicas must be greater than 0")
	}
	if s.MinReplicas < 0 {
		return fmt.Errorf("minReplicas cannot be negative")
	}
	if s.MinReplicas > s.MaxReplicas {
		return fmt.Errorf("minReplicas cannot be greater than maxReplicas")
	}

	// Validate metrics
	if err := s.Metrics.Validate(); err != nil {
		return fmt.Errorf("metrics validation failed: %w", err)
	}

	return nil
}

// Validate validates the MetricsSpec
func (m *MetricsSpec) Validate() error {
	hasEnabledMetric := false

	if m.Latency != nil && m.Latency.Enabled {
		hasEnabledMetric = true
		if m.Latency.TargetP99Ms <= 0 && m.Latency.TargetP95Ms <= 0 {
			return fmt.Errorf("latency metric enabled but no target specified")
		}
	}

	if m.GPUUtilization != nil && m.GPUUtilization.Enabled {
		hasEnabledMetric = true
		if m.GPUUtilization.TargetPercentage <= 0 || m.GPUUtilization.TargetPercentage > 100 {
			return fmt.Errorf("gpuUtilization.targetPercentage must be between 1 and 100")
		}
	}

	if m.RequestQueueDepth != nil && m.RequestQueueDepth.Enabled {
		hasEnabledMetric = true
		if m.RequestQueueDepth.TargetDepth < 0 {
			return fmt.Errorf("requestQueueDepth.targetDepth cannot be negative")
		}
	}

	if !hasEnabledMetric {
		return fmt.Errorf("at least one metric must be enabled")
	}

	return nil
}

// SetDefaults sets default values for the policy
func (p *AIInferenceAutoscalerPolicy) SetDefaults() {
	if p.Spec.MinReplicas == 0 {
		p.Spec.MinReplicas = 1
	}
	if p.Spec.CooldownPeriod == 0 {
		p.Spec.CooldownPeriod = 300
	}
	if p.Spec.TargetRef.APIVersion == "" {
		p.Spec.TargetRef.APIVersion = "apps/v1"
	}
}
