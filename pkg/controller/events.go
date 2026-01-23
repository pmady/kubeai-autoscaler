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

// Package controller provides event recording utilities for the autoscaler.
package controller

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	kubeaiv1alpha1 "github.com/pmady/kubeai-autoscaler/api/v1alpha1"
)

const (
	// ReasonScaledUp indicates the target was scaled up.
	ReasonScaledUp = "ScaledUp"
	// ReasonScaledDown indicates the target was scaled down.
	ReasonScaledDown = "ScaledDown"
	// ReasonScalingFailed indicates scaling operation failed.
	ReasonScalingFailed = "ScalingFailed"
	// ReasonMetricsFailed indicates metrics fetch failed.
	ReasonMetricsFailed = "MetricsFetchFailed"
	// ReasonTargetNotFound indicates the scale target was not found.
	ReasonTargetNotFound = "TargetNotFound"
	// ReasonCooldown indicates cooldown period is active.
	ReasonCooldown = "CooldownActive"
	// ReasonUnknownAlgorithm indicates the specified algorithm is not registered.
	ReasonUnknownAlgorithm = "UnknownAlgorithm"
)

// EventRecorder wraps the Kubernetes event recorder
type EventRecorder struct {
	recorder record.EventRecorder
}

// NewEventRecorder creates a new EventRecorder
func NewEventRecorder(recorder record.EventRecorder) *EventRecorder {
	return &EventRecorder{
		recorder: recorder,
	}
}

// RecordScaleUp records a scale up event
func (e *EventRecorder) RecordScaleUp(policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy, from, to int32) {
	if e.recorder == nil {
		return
	}
	e.recorder.Eventf(policy, corev1.EventTypeNormal, ReasonScaledUp,
		"Scaled %s/%s from %d to %d replicas",
		policy.Spec.TargetRef.Kind, policy.Spec.TargetRef.Name, from, to)
}

// RecordScaleDown records a scale down event
func (e *EventRecorder) RecordScaleDown(policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy, from, to int32) {
	if e.recorder == nil {
		return
	}
	e.recorder.Eventf(policy, corev1.EventTypeNormal, ReasonScaledDown,
		"Scaled %s/%s from %d to %d replicas",
		policy.Spec.TargetRef.Kind, policy.Spec.TargetRef.Name, from, to)
}

// RecordScalingFailed records a scaling failure event
func (e *EventRecorder) RecordScalingFailed(policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy, err error) {
	if e.recorder == nil {
		return
	}
	e.recorder.Eventf(policy, corev1.EventTypeWarning, ReasonScalingFailed,
		"Failed to scale %s/%s: %v",
		policy.Spec.TargetRef.Kind, policy.Spec.TargetRef.Name, err)
}

// RecordMetricsFailed records a metrics fetch failure event
func (e *EventRecorder) RecordMetricsFailed(policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy, err error) {
	if e.recorder == nil {
		return
	}
	e.recorder.Eventf(policy, corev1.EventTypeWarning, ReasonMetricsFailed,
		"Failed to fetch metrics: %v", err)
}

// RecordTargetNotFound records a target not found event
func (e *EventRecorder) RecordTargetNotFound(policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy, err error) {
	if e.recorder == nil {
		return
	}
	e.recorder.Eventf(policy, corev1.EventTypeWarning, ReasonTargetNotFound,
		"Target %s/%s not found: %v",
		policy.Spec.TargetRef.Kind, policy.Spec.TargetRef.Name, err)
}

// RecordCooldown records a cooldown active event
func (e *EventRecorder) RecordCooldown(policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy, remainingSeconds int) {
	if e.recorder == nil {
		return
	}
	e.recorder.Eventf(policy, corev1.EventTypeNormal, ReasonCooldown,
		"Scaling skipped, cooldown active for %d more seconds", remainingSeconds)
}

// RecordUnknownAlgorithm records a warning event when the specified algorithm is not found
func (e *EventRecorder) RecordUnknownAlgorithm(policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy, requested, fallback string, available []string) {
	if e.recorder == nil {
		return
	}
	e.recorder.Eventf(policy, corev1.EventTypeWarning, ReasonUnknownAlgorithm,
		"spec.algorithm.name=%q is not registered; falling back to %q. Available: %v",
		requested, fallback, available)
}
