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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetRef.name`
// +kubebuilder:printcolumn:name="Min",type=integer,JSONPath=`.spec.minReplicas`
// +kubebuilder:printcolumn:name="Max",type=integer,JSONPath=`.spec.maxReplicas`
// +kubebuilder:printcolumn:name="Current",type=integer,JSONPath=`.status.currentReplicas`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AIInferenceAutoscalerPolicy defines autoscaling rules for AI inference workloads
type AIInferenceAutoscalerPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AIInferenceAutoscalerPolicySpec   `json:"spec,omitempty"`
	Status AIInferenceAutoscalerPolicyStatus `json:"status,omitempty"`
}

// AIInferenceAutoscalerPolicySpec defines the desired state
type AIInferenceAutoscalerPolicySpec struct {
	// TargetRef references the target Deployment or StatefulSet
	TargetRef TargetRef `json:"targetRef"`

	// MinReplicas is the minimum number of replicas
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	MinReplicas int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the maximum number of replicas
	// +kubebuilder:validation:Minimum=1
	MaxReplicas int32 `json:"maxReplicas"`

	// CooldownPeriod is the cooldown period in seconds between scaling events
	// +kubebuilder:default=300
	// +kubebuilder:validation:Minimum=0
	CooldownPeriod int32 `json:"cooldownPeriod,omitempty"`

	// Metrics configuration for scaling decisions
	Metrics MetricsSpec `json:"metrics"`

	// ScaleUp behavior configuration
	// +optional
	ScaleUp *ScaleBehavior `json:"scaleUp,omitempty"`

	// ScaleDown behavior configuration
	// +optional
	ScaleDown *ScaleBehavior `json:"scaleDown,omitempty"`
}

// TargetRef references the target resource to scale
type TargetRef struct {
	// APIVersion of the target resource
	APIVersion string `json:"apiVersion"`

	// Kind of the target resource (Deployment or StatefulSet)
	// +kubebuilder:validation:Enum=Deployment;StatefulSet
	Kind string `json:"kind"`

	// Name of the target resource
	Name string `json:"name"`
}

// MetricsSpec defines the metrics configuration
type MetricsSpec struct {
	// Latency-based scaling configuration
	// +optional
	Latency *LatencyMetric `json:"latency,omitempty"`

	// GPU utilization-based scaling configuration
	// +optional
	GPUUtilization *GPUUtilizationMetric `json:"gpuUtilization,omitempty"`

	// Request queue depth-based scaling configuration
	// +optional
	RequestQueueDepth *QueueDepthMetric `json:"requestQueueDepth,omitempty"`
}

// LatencyMetric defines latency-based scaling
type LatencyMetric struct {
	// Enabled indicates if latency-based scaling is enabled
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// TargetP99Ms is the target P99 latency in milliseconds
	// +optional
	TargetP99Ms int32 `json:"targetP99Ms,omitempty"`

	// TargetP95Ms is the target P95 latency in milliseconds
	// +optional
	TargetP95Ms int32 `json:"targetP95Ms,omitempty"`

	// PrometheusQuery is a custom Prometheus query for latency metric
	// +optional
	PrometheusQuery string `json:"prometheusQuery,omitempty"`
}

// GPUUtilizationMetric defines GPU utilization-based scaling
type GPUUtilizationMetric struct {
	// Enabled indicates if GPU-based scaling is enabled
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// TargetPercentage is the target GPU utilization percentage
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	TargetPercentage int32 `json:"targetPercentage,omitempty"`

	// PrometheusQuery is a custom Prometheus query for GPU utilization
	// +optional
	PrometheusQuery string `json:"prometheusQuery,omitempty"`
}

// QueueDepthMetric defines queue depth-based scaling
type QueueDepthMetric struct {
	// Enabled indicates if queue depth-based scaling is enabled
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// TargetDepth is the target queue depth per replica
	// +kubebuilder:validation:Minimum=0
	TargetDepth int32 `json:"targetDepth,omitempty"`

	// PrometheusQuery is a custom Prometheus query for queue depth
	// +optional
	PrometheusQuery string `json:"prometheusQuery,omitempty"`
}

// ScaleBehavior defines scaling behavior
type ScaleBehavior struct {
	// StabilizationWindowSeconds is the stabilization window
	// +kubebuilder:validation:Minimum=0
	StabilizationWindowSeconds int32 `json:"stabilizationWindowSeconds,omitempty"`

	// Policies is a list of scaling policies
	// +optional
	Policies []ScalingPolicy `json:"policies,omitempty"`
}

// ScalingPolicy defines a scaling policy
type ScalingPolicy struct {
	// Type is the type of scaling policy (Pods or Percent)
	// +kubebuilder:validation:Enum=Pods;Percent
	Type string `json:"type"`

	// Value is the value for the policy
	Value int32 `json:"value"`

	// PeriodSeconds is the period for the policy
	PeriodSeconds int32 `json:"periodSeconds"`
}

// AIInferenceAutoscalerPolicyStatus defines the observed state
type AIInferenceAutoscalerPolicyStatus struct {
	// CurrentReplicas is the current number of replicas
	CurrentReplicas int32 `json:"currentReplicas,omitempty"`

	// DesiredReplicas is the desired number of replicas
	DesiredReplicas int32 `json:"desiredReplicas,omitempty"`

	// LastScaleTime is the last time the policy scaled the target
	// +optional
	LastScaleTime *metav1.Time `json:"lastScaleTime,omitempty"`

	// CurrentMetrics contains the current metric values
	// +optional
	CurrentMetrics *CurrentMetrics `json:"currentMetrics,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// CurrentMetrics contains current metric values
type CurrentMetrics struct {
	// LatencyP99Ms is the current P99 latency in milliseconds
	LatencyP99Ms int32 `json:"latencyP99Ms,omitempty"`

	// LatencyP95Ms is the current P95 latency in milliseconds
	LatencyP95Ms int32 `json:"latencyP95Ms,omitempty"`

	// GPUUtilizationPercent is the current GPU utilization percentage
	GPUUtilizationPercent int32 `json:"gpuUtilizationPercent,omitempty"`

	// RequestQueueDepth is the current request queue depth
	RequestQueueDepth int32 `json:"requestQueueDepth,omitempty"`
}

// +kubebuilder:object:root=true

// AIInferenceAutoscalerPolicyList contains a list of AIInferenceAutoscalerPolicy
type AIInferenceAutoscalerPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AIInferenceAutoscalerPolicy `json:"items"`
}
