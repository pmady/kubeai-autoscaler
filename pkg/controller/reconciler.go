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
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kubeaiv1alpha1 "github.com/pmady/kubeai-autoscaler/api/v1alpha1"
	"github.com/pmady/kubeai-autoscaler/pkg/metrics"
	"github.com/pmady/kubeai-autoscaler/pkg/scaling"
)

const (
	// ConditionTypeReady indicates the policy is ready
	ConditionTypeReady = "Ready"
	// ConditionTypeScaling indicates scaling is in progress
	ConditionTypeScaling = "Scaling"
	// ConditionTypeAlgorithmValid indicates the configured algorithm is valid
	ConditionTypeAlgorithmValid = "AlgorithmValid"
	// DefaultCooldownPeriod is the default cooldown between scaling events
	DefaultCooldownPeriod = 300 * time.Second
	// DefaultRequeueInterval is the default requeue interval
	DefaultRequeueInterval = 30 * time.Second
)

// DefaultAlgorithmName is the default scaling algorithm
const DefaultAlgorithmName = "MaxRatio"

// DefaultTolerance is the default tolerance for scaling algorithms
const DefaultTolerance = 0.1

// AIInferenceAutoscalerPolicyReconciler reconciles AIInferenceAutoscalerPolicy objects
type AIInferenceAutoscalerPolicyReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	MetricsClient     metrics.Client
	AlgorithmRegistry *scaling.Registry
	EventRecorder     *EventRecorder
	LastScaleTime     map[string]time.Time
	CooldownPeriod    time.Duration
}

// NewReconciler creates a new reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme, metricsClient metrics.Client, registry *scaling.Registry, eventRecorder *EventRecorder) *AIInferenceAutoscalerPolicyReconciler {
	if registry == nil {
		registry = scaling.DefaultRegistry
	}
	return &AIInferenceAutoscalerPolicyReconciler{
		Client:            client,
		Scheme:            scheme,
		MetricsClient:     metricsClient,
		AlgorithmRegistry: registry,
		EventRecorder:     eventRecorder,
		LastScaleTime:     make(map[string]time.Time),
		CooldownPeriod:    DefaultCooldownPeriod,
	}
}

// +kubebuilder:rbac:groups=kubeai.io,resources=aiinferenceautoscalerpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubeai.io,resources=aiinferenceautoscalerpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubeai.io,resources=aiinferenceautoscalerpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile handles the reconciliation loop for AIInferenceAutoscalerPolicy
func (r *AIInferenceAutoscalerPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the AIInferenceAutoscalerPolicy instance
	policy := &kubeaiv1alpha1.AIInferenceAutoscalerPolicy{}
	if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("AIInferenceAutoscalerPolicy not found, ignoring")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling AIInferenceAutoscalerPolicy",
		"name", policy.Name,
		"namespace", policy.Namespace,
		"target", policy.Spec.TargetRef.Name)

	// Get current replica count
	currentReplicas, err := r.getCurrentReplicas(ctx, policy)
	if err != nil {
		logger.Error(err, "Failed to get current replicas")
		r.updateCondition(ctx, policy, ConditionTypeReady, metav1.ConditionFalse, "TargetNotFound", err.Error())
		return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, nil
	}

	// Fetch current metrics
	currentMetrics, err := r.fetchMetrics(ctx, policy)
	if err != nil {
		logger.Error(err, "Failed to fetch metrics")
		r.updateCondition(ctx, policy, ConditionTypeReady, metav1.ConditionFalse, "MetricsFetchFailed", err.Error())
		return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, nil
	}

	// Calculate desired replicas
	desiredReplicas, algorithmUsed, scaleReason, algorithmNotFound, requestedAlgoName := r.calculateDesiredReplicas(ctx, policy, currentReplicas, currentMetrics)

	// Handle algorithm validity feedback
	if requestedAlgoName != "" {
		if algorithmNotFound {
			// Only emit event if condition is transitioning (prevent spam)
			if !r.hasCondition(policy, ConditionTypeAlgorithmValid, metav1.ConditionFalse, ReasonUnknownAlgorithm) {
				if r.EventRecorder != nil {
					r.EventRecorder.RecordUnknownAlgorithm(policy, requestedAlgoName, algorithmUsed, r.AlgorithmRegistry.List())
				}
			}
			r.updateCondition(ctx, policy, ConditionTypeAlgorithmValid, metav1.ConditionFalse,
				ReasonUnknownAlgorithm,
				fmt.Sprintf("Algorithm %q not found, using fallback %q", requestedAlgoName, algorithmUsed))
		} else {
			r.updateCondition(ctx, policy, ConditionTypeAlgorithmValid, metav1.ConditionTrue,
				"AlgorithmFound", fmt.Sprintf("Using algorithm %q", algorithmUsed))
		}
	}

	// Check cooldown period
	policyKey := fmt.Sprintf("%s/%s", policy.Namespace, policy.Name)
	if lastScale, ok := r.LastScaleTime[policyKey]; ok {
		cooldown := time.Duration(policy.Spec.CooldownPeriod) * time.Second
		if cooldown == 0 {
			cooldown = DefaultCooldownPeriod
		}
		if time.Since(lastScale) < cooldown && desiredReplicas != currentReplicas {
			logger.Info("Cooldown period not elapsed, skipping scaling",
				"lastScale", lastScale,
				"cooldown", cooldown)
			return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, nil
		}
	}

	// Scale if needed
	if desiredReplicas != currentReplicas {
		logger.Info("Scaling target",
			"current", currentReplicas,
			"desired", desiredReplicas,
			"algorithm", algorithmUsed,
			"reason", scaleReason)

		if err := r.scaleTarget(ctx, policy, desiredReplicas); err != nil {
			logger.Error(err, "Failed to scale target")
			r.updateCondition(ctx, policy, ConditionTypeScaling, metav1.ConditionFalse, "ScaleFailed", err.Error())
			return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, nil
		}

		r.LastScaleTime[policyKey] = time.Now()
		r.updateCondition(ctx, policy, ConditionTypeScaling, metav1.ConditionTrue, "Scaled",
			fmt.Sprintf("Scaled from %d to %d replicas using %s algorithm", currentReplicas, desiredReplicas, algorithmUsed))
	}

	// Update status
	if err := r.updateStatus(ctx, policy, currentReplicas, desiredReplicas, currentMetrics, algorithmUsed, scaleReason); err != nil {
		logger.Error(err, "Failed to update status")
	}

	r.updateCondition(ctx, policy, ConditionTypeReady, metav1.ConditionTrue, "Ready", "Policy is active")

	return ctrl.Result{RequeueAfter: DefaultRequeueInterval}, nil
}

// getCurrentReplicas gets the current replica count from the target
func (r *AIInferenceAutoscalerPolicyReconciler) getCurrentReplicas(ctx context.Context, policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy) (int32, error) {
	switch policy.Spec.TargetRef.Kind {
	case "Deployment":
		deployment := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: policy.Namespace,
			Name:      policy.Spec.TargetRef.Name,
		}, deployment); err != nil {
			return 0, err
		}
		if deployment.Spec.Replicas == nil {
			return 1, nil
		}
		return *deployment.Spec.Replicas, nil

	case "StatefulSet":
		statefulSet := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: policy.Namespace,
			Name:      policy.Spec.TargetRef.Name,
		}, statefulSet); err != nil {
			return 0, err
		}
		if statefulSet.Spec.Replicas == nil {
			return 1, nil
		}
		return *statefulSet.Spec.Replicas, nil

	default:
		return 0, fmt.Errorf("unsupported target kind: %s", policy.Spec.TargetRef.Kind)
	}
}

// fetchMetrics fetches current metrics from Prometheus
func (r *AIInferenceAutoscalerPolicyReconciler) fetchMetrics(ctx context.Context, policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy) (*kubeaiv1alpha1.CurrentMetrics, error) {
	currentMetrics := &kubeaiv1alpha1.CurrentMetrics{}

	if r.MetricsClient == nil {
		return currentMetrics, nil
	}

	// Fetch latency metrics
	if policy.Spec.Metrics.Latency != nil && policy.Spec.Metrics.Latency.Enabled {
		if policy.Spec.Metrics.Latency.TargetP99Ms > 0 {
			latency, err := r.MetricsClient.GetLatencyP99(ctx, policy.Spec.Metrics.Latency.PrometheusQuery)
			if err == nil {
				currentMetrics.LatencyP99Ms = int32(latency * 1000) // Convert to ms
			}
		}
		if policy.Spec.Metrics.Latency.TargetP95Ms > 0 {
			latency, err := r.MetricsClient.GetLatencyP95(ctx, policy.Spec.Metrics.Latency.PrometheusQuery)
			if err == nil {
				currentMetrics.LatencyP95Ms = int32(latency * 1000) // Convert to ms
			}
		}
	}

	// Fetch GPU utilization
	if policy.Spec.Metrics.GPUUtilization != nil && policy.Spec.Metrics.GPUUtilization.Enabled {
		gpu, err := r.MetricsClient.GetGPUUtilization(ctx, policy.Spec.Metrics.GPUUtilization.PrometheusQuery)
		if err == nil {
			currentMetrics.GPUUtilizationPercent = int32(gpu)
		}
	}

	// Fetch queue depth
	if policy.Spec.Metrics.RequestQueueDepth != nil && policy.Spec.Metrics.RequestQueueDepth.Enabled {
		depth, err := r.MetricsClient.GetQueueDepth(ctx, policy.Spec.Metrics.RequestQueueDepth.PrometheusQuery)
		if err == nil {
			currentMetrics.RequestQueueDepth = int32(depth) // #nosec G115 - queue depth won't exceed int32 max in practice
		}
	}

	return currentMetrics, nil
}

// calculateDesiredReplicas computes the desired replica count based on metrics.
// Returns:
//   - desiredReplicas: the computed replica count
//   - algorithmUsed: the name of the algorithm that was actually used
//   - reason: explanation of the scaling decision
//   - requestedAlgorithmNotFound: true if the user-specified algorithm was not found
//   - requestedName: the algorithm name the user specified (empty if none specified)
func (r *AIInferenceAutoscalerPolicyReconciler) calculateDesiredReplicas(
	ctx context.Context,
	policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy,
	currentReplicas int32,
	currentMetrics *kubeaiv1alpha1.CurrentMetrics,
) (desiredReplicas int32, algorithmUsed string, reason string, requestedAlgorithmNotFound bool, requestedName string) {
	logger := log.FromContext(ctx)

	// Determine which algorithm to use
	algorithmName := DefaultAlgorithmName
	tolerance := DefaultTolerance
	var weights []float64

	if policy.Spec.Algorithm != nil {
		if policy.Spec.Algorithm.Name != "" {
			requestedName = policy.Spec.Algorithm.Name
			algorithmName = policy.Spec.Algorithm.Name
		}
		// Always honor the configured tolerance, including 0 (zero tolerance)
		tolerance = policy.Spec.Algorithm.Tolerance
		weights = policy.Spec.Algorithm.Weights
	}

	// Get the algorithm from registry
	algorithm, err := r.AlgorithmRegistry.Get(algorithmName)
	if err != nil {
		logger.Error(err, "Algorithm not found, falling back to default", "algorithm", algorithmName)

		// Only flag as not found if user explicitly specified an algorithm
		if requestedName != "" {
			requestedAlgorithmNotFound = true
		}

		// Try the default algorithm in the configured registry first.
		algorithmName = DefaultAlgorithmName
		algorithm, err = r.AlgorithmRegistry.Get(DefaultAlgorithmName)
		if err != nil {
			logger.Error(err, "Default algorithm not found in custom registry, trying global default registry", "algorithm", DefaultAlgorithmName)

			// As a final fallback, try the global default registry.
			algorithm, err = scaling.DefaultRegistry.Get(DefaultAlgorithmName)
		}

		// If we still don't have a valid algorithm, keep the current replicas to avoid a panic.
		if err != nil || algorithm == nil {
			logger.Error(err, "No valid scaling algorithm available, keeping current replicas", "algorithm", algorithmName)
			return currentReplicas, algorithmName, "no algorithm available", requestedAlgorithmNotFound, requestedName
		}
	}

	// If using WeightedRatio, set the weights on a per-request copy to avoid mutating shared instances
	if weightedAlgo, ok := algorithm.(*scaling.WeightedRatioAlgorithm); ok && len(weights) > 0 {
		algoCopy := *weightedAlgo
		copyPtr := &algoCopy
		copyPtr.SetWeights(weights)
		algorithm = copyPtr
	}

	// Build metric ratios
	metricRatios := r.buildMetricRatios(policy, currentReplicas, currentMetrics)

	// Apply min/max constraints
	minReplicas := policy.Spec.MinReplicas
	if minReplicas == 0 {
		minReplicas = 1
	}
	maxReplicas := policy.Spec.MaxReplicas

	// Build scaling input
	input := scaling.ScalingInput{
		CurrentReplicas: currentReplicas,
		MinReplicas:     minReplicas,
		MaxReplicas:     maxReplicas,
		MetricRatios:    metricRatios,
		Tolerance:       tolerance,
		PolicyName:      policy.Name,
		PolicyNamespace: policy.Namespace,
	}

	// Compute scale using the algorithm
	result, err := algorithm.ComputeScale(ctx, input)
	if err != nil {
		logger.Error(err, "Algorithm computation failed, keeping current replicas", "algorithm", algorithmName)
		return currentReplicas, algorithmName, "computation failed", requestedAlgorithmNotFound, requestedName
	}

	logger.Info("Calculated desired replicas",
		"algorithm", algorithmName,
		"current", currentReplicas,
		"desired", result.DesiredReplicas,
		"reason", result.Reason,
		"tolerance", tolerance,
		"min", minReplicas,
		"max", maxReplicas)

	return result.DesiredReplicas, algorithmName, result.Reason, requestedAlgorithmNotFound, requestedName
}

// buildMetricRatios builds the list of metric ratios from current metrics
func (r *AIInferenceAutoscalerPolicyReconciler) buildMetricRatios(
	policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy,
	currentReplicas int32,
	currentMetrics *kubeaiv1alpha1.CurrentMetrics,
) []float64 {
	var ratios []float64

	// Calculate latency ratios
	if policy.Spec.Metrics.Latency != nil && policy.Spec.Metrics.Latency.Enabled {
		if policy.Spec.Metrics.Latency.TargetP99Ms > 0 && currentMetrics.LatencyP99Ms > 0 {
			ratio := float64(currentMetrics.LatencyP99Ms) / float64(policy.Spec.Metrics.Latency.TargetP99Ms)
			ratios = append(ratios, ratio)
		}
		if policy.Spec.Metrics.Latency.TargetP95Ms > 0 && currentMetrics.LatencyP95Ms > 0 {
			ratio := float64(currentMetrics.LatencyP95Ms) / float64(policy.Spec.Metrics.Latency.TargetP95Ms)
			ratios = append(ratios, ratio)
		}
	}

	// Calculate GPU utilization ratio
	if policy.Spec.Metrics.GPUUtilization != nil && policy.Spec.Metrics.GPUUtilization.Enabled {
		if policy.Spec.Metrics.GPUUtilization.TargetPercentage > 0 && currentMetrics.GPUUtilizationPercent > 0 {
			ratio := float64(currentMetrics.GPUUtilizationPercent) / float64(policy.Spec.Metrics.GPUUtilization.TargetPercentage)
			ratios = append(ratios, ratio)
		}
	}

	// Calculate queue depth ratio
	if policy.Spec.Metrics.RequestQueueDepth != nil && policy.Spec.Metrics.RequestQueueDepth.Enabled {
		if policy.Spec.Metrics.RequestQueueDepth.TargetDepth > 0 && currentMetrics.RequestQueueDepth > 0 {
			ratio := float64(currentMetrics.RequestQueueDepth) / float64(policy.Spec.Metrics.RequestQueueDepth.TargetDepth*currentReplicas)
			ratios = append(ratios, ratio)
		}
	}

	return ratios
}

// scaleTarget scales the target deployment or statefulset
func (r *AIInferenceAutoscalerPolicyReconciler) scaleTarget(ctx context.Context, policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy, replicas int32) error {
	switch policy.Spec.TargetRef.Kind {
	case "Deployment":
		deployment := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: policy.Namespace,
			Name:      policy.Spec.TargetRef.Name,
		}, deployment); err != nil {
			return err
		}
		deployment.Spec.Replicas = &replicas
		return r.Update(ctx, deployment)

	case "StatefulSet":
		statefulSet := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: policy.Namespace,
			Name:      policy.Spec.TargetRef.Name,
		}, statefulSet); err != nil {
			return err
		}
		statefulSet.Spec.Replicas = &replicas
		return r.Update(ctx, statefulSet)

	default:
		return fmt.Errorf("unsupported target kind: %s", policy.Spec.TargetRef.Kind)
	}
}

// updateStatus updates the policy status
func (r *AIInferenceAutoscalerPolicyReconciler) updateStatus(
	ctx context.Context,
	policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy,
	currentReplicas, desiredReplicas int32,
	currentMetrics *kubeaiv1alpha1.CurrentMetrics,
	algorithmUsed, scaleReason string,
) error {
	policy.Status.CurrentReplicas = currentReplicas
	policy.Status.DesiredReplicas = desiredReplicas
	policy.Status.CurrentMetrics = currentMetrics
	policy.Status.LastAlgorithm = algorithmUsed
	policy.Status.LastScaleReason = scaleReason

	if currentReplicas != desiredReplicas {
		now := metav1.Now()
		policy.Status.LastScaleTime = &now
	}

	return r.Status().Update(ctx, policy)
}

// updateCondition updates a condition on the policy
func (r *AIInferenceAutoscalerPolicyReconciler) updateCondition(
	ctx context.Context,
	policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy,
	conditionType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Find and update existing condition or append new one
	found := false
	for i, c := range policy.Status.Conditions {
		if c.Type == conditionType {
			policy.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		policy.Status.Conditions = append(policy.Status.Conditions, condition)
	}

	if err := r.Status().Update(ctx, policy); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update condition")
	}
}

// hasCondition checks if the policy already has a condition with the specified type, status, and reason
func (r *AIInferenceAutoscalerPolicyReconciler) hasCondition(
	policy *kubeaiv1alpha1.AIInferenceAutoscalerPolicy,
	conditionType string,
	status metav1.ConditionStatus,
	reason string,
) bool {
	for _, c := range policy.Status.Conditions {
		if c.Type == conditionType && c.Status == status && c.Reason == reason {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager
func (r *AIInferenceAutoscalerPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubeaiv1alpha1.AIInferenceAutoscalerPolicy{}).
		Complete(r)
}
