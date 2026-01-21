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
	"math"
	"sync"
	"time"

	kubeaiv1alpha1 "github.com/pmady/kubeai-autoscaler/api/v1alpha1"
	"github.com/pmady/kubeai-autoscaler/pkg/metrics"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// DefaultRequeueInterval is the default interval for requeuing reconciliation
	DefaultRequeueInterval = 30 * time.Second
	// DefaultCooldownPeriod is the default cooldown period between scaling events
	DefaultCooldownPeriod = 5 * time.Minute
)

// AIInferenceAutoscalerPolicyReconciler reconciles AIInferenceAutoscalerPolicy objects
type AIInferenceAutoscalerPolicyReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	MetricsClient  metrics.Client
	lastScaleTime  map[string]time.Time
	lastScaleMutex sync.RWMutex
}

// NewAIInferenceAutoscalerPolicyReconciler creates a new reconciler
func NewAIInferenceAutoscalerPolicyReconciler(
	client client.Client,
	scheme *runtime.Scheme,
	metricsClient metrics.Client,
) *AIInferenceAutoscalerPolicyReconciler {
	return &AIInferenceAutoscalerPolicyReconciler{
		Client:        client,
		Scheme:        scheme,
		MetricsClient: metricsClient,
		lastScaleTime: make(map[string]time.Time),
	}
}

// Reconcile handles the reconciliation loop for AIInferenceAutoscalerPolicy
func (r *AIInferenceAutoscalerPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling AIInferenceAutoscalerPolicy", "name", req.Name, "namespace", req.Namespace)

	// Fetch the AIInferenceAutoscalerPolicy instance
	policy := &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{}
	if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("AIInferenceAutoscalerPolicy not found, ignoring")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get AIInferenceAutoscalerPolicy")
		return ctrl.Result{}, err
	}

	// Get current replica count from target
	currentReplicas, err := r.getCurrentReplicas(ctx, policy)
	if err != nil {
		logger.Error(err, "Failed to get current replicas")
		return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, err
	}

	// Fetch current metrics from Prometheus
	currentMetrics, err := r.fetchMetrics(ctx, policy)
	if err != nil {
		logger.Error(err, "Failed to fetch metrics")
		// Continue with scaling decision based on available data
	}

	// Calculate desired replicas based on metrics and policy
	desiredReplicas := r.calculateDesiredReplicas(ctx, policy, currentReplicas, currentMetrics)

	// Check cooldown period
	policyKey := fmt.Sprintf("%s/%s", policy.Namespace, policy.Name)
	if !r.canScale(policyKey, policy.Spec.CooldownPeriod) {
		logger.Info("Cooldown period not elapsed, skipping scaling",
			"lastScaleTime", r.getLastScaleTime(policyKey))
		return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, nil
	}

	// Scale if needed
	if desiredReplicas != currentReplicas {
		logger.Info("Scaling target",
			"current", currentReplicas,
			"desired", desiredReplicas,
			"target", policy.Spec.TargetRef.Name)

		if err := r.scaleTarget(ctx, policy, desiredReplicas); err != nil {
			logger.Error(err, "Failed to scale target")
			return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, err
		}

		r.setLastScaleTime(policyKey, time.Now())
	}

	// Update status
	if err := r.updateStatus(ctx, policy, currentReplicas, desiredReplicas, currentMetrics); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, err
	}

	return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *AIInferenceAutoscalerPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubeaiv1alpha1.AIInferenceAutoscalerPolicy{}).
		Complete(r)
}

// fetchMetrics fetches current metrics from Prometheus
func (r *AIInferenceAutoscalerPolicyReconciler) fetchMetrics(
	ctx context.Context,
	policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy,
) (*kubeaiv1alpha1.CurrentMetrics, error) {
	logger := log.FromContext(ctx)
	currentMetrics := &kubeaiv1alpha1.CurrentMetrics{}

	if r.MetricsClient == nil {
		logger.Info("Metrics client not configured, skipping metrics fetch")
		return currentMetrics, nil
	}

	// Fetch latency metrics
	if policy.Spec.Metrics.Latency != nil && policy.Spec.Metrics.Latency.Enabled {
		query := policy.Spec.Metrics.Latency.PrometheusQuery
		if policy.Spec.Metrics.Latency.TargetP99Ms > 0 {
			latencyP99, err := r.MetricsClient.GetLatencyP99(ctx, query)
			if err != nil {
				logger.Error(err, "Failed to fetch P99 latency")
			} else {
				currentMetrics.LatencyP99Ms = int32(latencyP99 * 1000) // Convert to ms
			}
		}
		if policy.Spec.Metrics.Latency.TargetP95Ms > 0 {
			latencyP95, err := r.MetricsClient.GetLatencyP95(ctx, query)
			if err != nil {
				logger.Error(err, "Failed to fetch P95 latency")
			} else {
				currentMetrics.LatencyP95Ms = int32(latencyP95 * 1000) // Convert to ms
			}
		}
	}

	// Fetch GPU utilization
	if policy.Spec.Metrics.GPUUtilization != nil && policy.Spec.Metrics.GPUUtilization.Enabled {
		query := policy.Spec.Metrics.GPUUtilization.PrometheusQuery
		gpuUtil, err := r.MetricsClient.GetGPUUtilization(ctx, query)
		if err != nil {
			logger.Error(err, "Failed to fetch GPU utilization")
		} else {
			currentMetrics.GPUUtilizationPercent = int32(gpuUtil)
		}
	}

	// Fetch queue depth
	if policy.Spec.Metrics.RequestQueueDepth != nil && policy.Spec.Metrics.RequestQueueDepth.Enabled {
		query := policy.Spec.Metrics.RequestQueueDepth.PrometheusQuery
		queueDepth, err := r.MetricsClient.GetQueueDepth(ctx, query)
		if err != nil {
			logger.Error(err, "Failed to fetch queue depth")
		} else {
			currentMetrics.RequestQueueDepth = int32(queueDepth)
		}
	}

	return currentMetrics, nil
}

// calculateDesiredReplicas computes the desired replica count based on metrics
func (r *AIInferenceAutoscalerPolicyReconciler) calculateDesiredReplicas(
	ctx context.Context,
	policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy,
	currentReplicas int32,
	currentMetrics *kubeaiv1alpha1.CurrentMetrics,
) int32 {
	logger := log.FromContext(ctx)

	if currentMetrics == nil {
		return currentReplicas
	}

	maxRatio := 1.0

	// Calculate latency-based scaling ratio
	if policy.Spec.Metrics.Latency != nil && policy.Spec.Metrics.Latency.Enabled {
		if policy.Spec.Metrics.Latency.TargetP99Ms > 0 && currentMetrics.LatencyP99Ms > 0 {
			latencyRatio := float64(currentMetrics.LatencyP99Ms) / float64(policy.Spec.Metrics.Latency.TargetP99Ms)
			maxRatio = math.Max(maxRatio, latencyRatio)
			logger.V(1).Info("Latency P99 ratio", "current", currentMetrics.LatencyP99Ms,
				"target", policy.Spec.Metrics.Latency.TargetP99Ms, "ratio", latencyRatio)
		}
		if policy.Spec.Metrics.Latency.TargetP95Ms > 0 && currentMetrics.LatencyP95Ms > 0 {
			latencyRatio := float64(currentMetrics.LatencyP95Ms) / float64(policy.Spec.Metrics.Latency.TargetP95Ms)
			maxRatio = math.Max(maxRatio, latencyRatio)
			logger.V(1).Info("Latency P95 ratio", "current", currentMetrics.LatencyP95Ms,
				"target", policy.Spec.Metrics.Latency.TargetP95Ms, "ratio", latencyRatio)
		}
	}

	// Calculate GPU-based scaling ratio
	if policy.Spec.Metrics.GPUUtilization != nil && policy.Spec.Metrics.GPUUtilization.Enabled {
		if policy.Spec.Metrics.GPUUtilization.TargetPercentage > 0 && currentMetrics.GPUUtilizationPercent > 0 {
			gpuRatio := float64(currentMetrics.GPUUtilizationPercent) / float64(policy.Spec.Metrics.GPUUtilization.TargetPercentage)
			maxRatio = math.Max(maxRatio, gpuRatio)
			logger.V(1).Info("GPU utilization ratio", "current", currentMetrics.GPUUtilizationPercent,
				"target", policy.Spec.Metrics.GPUUtilization.TargetPercentage, "ratio", gpuRatio)
		}
	}

	// Calculate queue depth-based scaling ratio
	if policy.Spec.Metrics.RequestQueueDepth != nil && policy.Spec.Metrics.RequestQueueDepth.Enabled {
		if policy.Spec.Metrics.RequestQueueDepth.TargetDepth > 0 && currentMetrics.RequestQueueDepth > 0 {
			queueRatio := float64(currentMetrics.RequestQueueDepth) / float64(policy.Spec.Metrics.RequestQueueDepth.TargetDepth*currentReplicas)
			maxRatio = math.Max(maxRatio, queueRatio)
			logger.V(1).Info("Queue depth ratio", "current", currentMetrics.RequestQueueDepth,
				"targetPerReplica", policy.Spec.Metrics.RequestQueueDepth.TargetDepth, "ratio", queueRatio)
		}
	}

	// Calculate desired replicas
	desiredReplicas := int32(math.Ceil(float64(currentReplicas) * maxRatio))

	// Clamp to min/max
	minReplicas := policy.Spec.MinReplicas
	if minReplicas == 0 {
		minReplicas = 1
	}
	maxReplicas := policy.Spec.MaxReplicas

	if desiredReplicas < minReplicas {
		desiredReplicas = minReplicas
	}
	if desiredReplicas > maxReplicas {
		desiredReplicas = maxReplicas
	}

	logger.Info("Calculated desired replicas",
		"current", currentReplicas,
		"desired", desiredReplicas,
		"min", minReplicas,
		"max", maxReplicas,
		"scalingRatio", maxRatio)

	return desiredReplicas
}

// getCurrentReplicas gets the current replica count from the target
func (r *AIInferenceAutoscalerPolicyReconciler) getCurrentReplicas(
	ctx context.Context,
	policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy,
) (int32, error) {
	switch policy.Spec.TargetRef.Kind {
	case "Deployment":
		deployment := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: policy.Namespace,
			Name:      policy.Spec.TargetRef.Name,
		}, deployment); err != nil {
			return 0, fmt.Errorf("failed to get deployment: %w", err)
		}
		if deployment.Spec.Replicas != nil {
			return *deployment.Spec.Replicas, nil
		}
		return 1, nil

	case "StatefulSet":
		statefulSet := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: policy.Namespace,
			Name:      policy.Spec.TargetRef.Name,
		}, statefulSet); err != nil {
			return 0, fmt.Errorf("failed to get statefulset: %w", err)
		}
		if statefulSet.Spec.Replicas != nil {
			return *statefulSet.Spec.Replicas, nil
		}
		return 1, nil

	default:
		return 0, fmt.Errorf("unsupported target kind: %s", policy.Spec.TargetRef.Kind)
	}
}

// scaleTarget scales the target to the desired replicas
func (r *AIInferenceAutoscalerPolicyReconciler) scaleTarget(
	ctx context.Context,
	policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy,
	replicas int32,
) error {
	switch policy.Spec.TargetRef.Kind {
	case "Deployment":
		return r.scaleDeployment(ctx, policy.Namespace, policy.Spec.TargetRef.Name, replicas)
	case "StatefulSet":
		return r.scaleStatefulSet(ctx, policy.Namespace, policy.Spec.TargetRef.Name, replicas)
	default:
		return fmt.Errorf("unsupported target kind: %s", policy.Spec.TargetRef.Kind)
	}
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

	if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == replicas {
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

// scaleStatefulSet scales the target statefulset to the desired replicas
func (r *AIInferenceAutoscalerPolicyReconciler) scaleStatefulSet(
	ctx context.Context,
	namespace string,
	name string,
	replicas int32,
) error {
	logger := log.FromContext(ctx)

	statefulSet := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, statefulSet); err != nil {
		return fmt.Errorf("failed to get statefulset: %w", err)
	}

	if statefulSet.Spec.Replicas != nil && *statefulSet.Spec.Replicas == replicas {
		logger.Info("StatefulSet already at desired replicas", "replicas", replicas)
		return nil
	}

	statefulSet.Spec.Replicas = &replicas
	if err := r.Update(ctx, statefulSet); err != nil {
		return fmt.Errorf("failed to update statefulset: %w", err)
	}

	logger.Info("Scaled statefulset", "name", name, "replicas", replicas)
	return nil
}

// updateStatus updates the policy status
func (r *AIInferenceAutoscalerPolicyReconciler) updateStatus(
	ctx context.Context,
	policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy,
	currentReplicas int32,
	desiredReplicas int32,
	currentMetrics *kubeaiv1alpha1.CurrentMetrics,
) error {
	policy.Status.CurrentReplicas = currentReplicas
	policy.Status.DesiredReplicas = desiredReplicas
	policy.Status.CurrentMetrics = currentMetrics

	if currentReplicas != desiredReplicas {
		now := metav1.Now()
		policy.Status.LastScaleTime = &now
	}

	return r.Status().Update(ctx, policy)
}

// canScale checks if the cooldown period has elapsed
func (r *AIInferenceAutoscalerPolicyReconciler) canScale(policyKey string, cooldownSeconds int32) bool {
	r.lastScaleMutex.RLock()
	defer r.lastScaleMutex.RUnlock()

	lastScale, exists := r.lastScaleTime[policyKey]
	if !exists {
		return true
	}

	cooldown := time.Duration(cooldownSeconds) * time.Second
	if cooldown == 0 {
		cooldown = DefaultCooldownPeriod
	}

	return time.Since(lastScale) >= cooldown
}

// getLastScaleTime returns the last scale time for a policy
func (r *AIInferenceAutoscalerPolicyReconciler) getLastScaleTime(policyKey string) time.Time {
	r.lastScaleMutex.RLock()
	defer r.lastScaleMutex.RUnlock()
	return r.lastScaleTime[policyKey]
}

// setLastScaleTime sets the last scale time for a policy
func (r *AIInferenceAutoscalerPolicyReconciler) setLastScaleTime(policyKey string, t time.Time) {
	r.lastScaleMutex.Lock()
	defer r.lastScaleMutex.Unlock()
	r.lastScaleTime[policyKey] = t
}
