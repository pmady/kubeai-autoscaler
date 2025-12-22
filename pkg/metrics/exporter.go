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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ScalingDecisions tracks the number of scaling decisions made
	ScalingDecisions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_autoscaler_scaling_decisions_total",
			Help: "Total number of scaling decisions made by the autoscaler",
		},
		[]string{"namespace", "policy", "direction"}, // direction: up, down, none
	)

	// CurrentReplicas tracks the current replica count for each policy
	CurrentReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeai_autoscaler_current_replicas",
			Help: "Current number of replicas for the target workload",
		},
		[]string{"namespace", "policy", "target"},
	)

	// DesiredReplicas tracks the desired replica count for each policy
	DesiredReplicas = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeai_autoscaler_desired_replicas",
			Help: "Desired number of replicas for the target workload",
		},
		[]string{"namespace", "policy", "target"},
	)

	// MetricValue tracks the current metric values
	MetricValue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeai_autoscaler_metric_value",
			Help: "Current value of the metric being used for scaling",
		},
		[]string{"namespace", "policy", "metric_type"},
	)

	// MetricTarget tracks the target metric values
	MetricTarget = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeai_autoscaler_metric_target",
			Help: "Target value of the metric being used for scaling",
		},
		[]string{"namespace", "policy", "metric_type"},
	)

	// ReconcileLatency tracks the latency of reconciliation loops
	ReconcileLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeai_autoscaler_reconcile_duration_seconds",
			Help:    "Duration of reconciliation loops in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"namespace", "policy"},
	)

	// ReconcileErrors tracks reconciliation errors
	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeai_autoscaler_reconcile_errors_total",
			Help: "Total number of reconciliation errors",
		},
		[]string{"namespace", "policy", "error_type"},
	)

	// CooldownActive tracks whether cooldown is active for a policy
	CooldownActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeai_autoscaler_cooldown_active",
			Help: "Whether cooldown is currently active (1) or not (0)",
		},
		[]string{"namespace", "policy"},
	)

	// LastScaleTime tracks the timestamp of the last scaling event
	LastScaleTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubeai_autoscaler_last_scale_time_seconds",
			Help: "Unix timestamp of the last scaling event",
		},
		[]string{"namespace", "policy"},
	)
)

func init() {
	// Register metrics with the controller-runtime metrics registry
	metrics.Registry.MustRegister(
		ScalingDecisions,
		CurrentReplicas,
		DesiredReplicas,
		MetricValue,
		MetricTarget,
		ReconcileLatency,
		ReconcileErrors,
		CooldownActive,
		LastScaleTime,
	)
}

// RecordScalingDecision records a scaling decision metric
func RecordScalingDecision(namespace, policy, direction string) {
	ScalingDecisions.WithLabelValues(namespace, policy, direction).Inc()
}

// RecordReplicaCounts records current and desired replica counts
func RecordReplicaCounts(namespace, policy, target string, current, desired int32) {
	CurrentReplicas.WithLabelValues(namespace, policy, target).Set(float64(current))
	DesiredReplicas.WithLabelValues(namespace, policy, target).Set(float64(desired))
}

// RecordMetricValues records current metric values and targets
func RecordMetricValues(namespace, policy, metricType string, value, target float64) {
	MetricValue.WithLabelValues(namespace, policy, metricType).Set(value)
	MetricTarget.WithLabelValues(namespace, policy, metricType).Set(target)
}

// RecordReconcileLatency records the duration of a reconciliation loop
func RecordReconcileLatency(namespace, policy string, durationSeconds float64) {
	ReconcileLatency.WithLabelValues(namespace, policy).Observe(durationSeconds)
}

// RecordReconcileError records a reconciliation error
func RecordReconcileError(namespace, policy, errorType string) {
	ReconcileErrors.WithLabelValues(namespace, policy, errorType).Inc()
}

// RecordCooldownStatus records whether cooldown is active
func RecordCooldownStatus(namespace, policy string, active bool) {
	value := 0.0
	if active {
		value = 1.0
	}
	CooldownActive.WithLabelValues(namespace, policy).Set(value)
}

// RecordLastScaleTime records the timestamp of the last scaling event
func RecordLastScaleTime(namespace, policy string, timestamp float64) {
	LastScaleTime.WithLabelValues(namespace, policy).Set(timestamp)
}
