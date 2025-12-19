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
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AIInferenceAutoscalerPolicyReconciler reconciles AIInferenceAutoscalerPolicy objects
type AIInferenceAutoscalerPolicyReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	MetricsClient   MetricsClient
	LastScaleTime   map[string]time.Time
	CooldownPeriod  time.Duration
}

// MetricsClient interface for fetching metrics from Prometheus
type MetricsClient interface {
	GetLatencyP99(ctx context.Context, query string) (float64, error)
	GetLatencyP95(ctx context.Context, query string) (float64, error)
	GetGPUUtilization(ctx context.Context, query string) (float64, error)
	GetQueueDepth(ctx context.Context, query string) (int64, error)
}

// Reconcile handles the reconciliation loop for AIInferenceAutoscalerPolicy
func (r *AIInferenceAutoscalerPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// TODO: Fetch the AIInferenceAutoscalerPolicy instance
	// policy := &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{}
	// if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }

	logger.Info("Reconciling AIInferenceAutoscalerPolicy", "name", req.Name, "namespace", req.Namespace)

	// TODO: Implement reconciliation logic
	// 1. Fetch current metrics from Prometheus
	// 2. Calculate desired replicas based on metrics and policy
	// 3. Check cooldown period
	// 4. Scale the target deployment/statefulset
	// 5. Update status

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *AIInferenceAutoscalerPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// For(&kubeaiv1alpha1.AIInferenceAutoscalerPolicy{}).
		Complete(r)
}

// calculateDesiredReplicas computes the desired replica count based on metrics
func (r *AIInferenceAutoscalerPolicyReconciler) calculateDesiredReplicas(
	ctx context.Context,
	currentReplicas int32,
	minReplicas int32,
	maxReplicas int32,
	latencyTarget float64,
	gpuTarget float64,
	queueTarget int64,
) (int32, error) {
	logger := log.FromContext(ctx)

	// Fetch current metrics
	// currentLatency, err := r.MetricsClient.GetLatencyP99(ctx, "")
	// currentGPU, err := r.MetricsClient.GetGPUUtilization(ctx, "")
	// currentQueue, err := r.MetricsClient.GetQueueDepth(ctx, "")

	// Calculate scaling ratio for each metric
	// latencyRatio := currentLatency / latencyTarget
	// gpuRatio := currentGPU / gpuTarget
	// queueRatio := float64(currentQueue) / float64(queueTarget)

	// Use the highest ratio to determine scaling
	// maxRatio := max(latencyRatio, gpuRatio, queueRatio)

	// Calculate desired replicas
	// desiredReplicas := int32(math.Ceil(float64(currentReplicas) * maxRatio))

	// Clamp to min/max
	// desiredReplicas = max(minReplicas, min(maxReplicas, desiredReplicas))

	logger.Info("Calculated desired replicas",
		"current", currentReplicas,
		"min", minReplicas,
		"max", maxReplicas,
	)

	return currentReplicas, nil
}

// scaleDeployment scales the target deployment to the desired replicas
func (r *AIInferenceAutoscalerPolicyReconciler) scaleDeployment(
	ctx context.Context,
	namespace string,
	name string,
	replicas int32,
) error {
	logger := log.FromContext(ctx)

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, deployment); err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	if *deployment.Spec.Replicas == replicas {
		logger.Info("Deployment already at desired replicas", "replicas", replicas)
		return nil
	}

	deployment.Spec.Replicas = &replicas
	if err := r.Update(ctx, deployment); err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	logger.Info("Scaled deployment", "name", name, "replicas", replicas)
	return nil
}
