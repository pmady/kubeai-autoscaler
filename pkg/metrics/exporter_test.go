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
	"testing"
)

func TestRecordScalingDecision(t *testing.T) {
	// Test that recording doesn't panic
	RecordScalingDecision("default", "test-policy", "up")
	RecordScalingDecision("default", "test-policy", "down")
	RecordScalingDecision("default", "test-policy", "none")
}

func TestRecordReplicaCounts(t *testing.T) {
	RecordReplicaCounts("default", "test-policy", "test-deployment", 2, 4)
}

func TestRecordMetricValues(t *testing.T) {
	RecordMetricValues("default", "test-policy", "latency_p99", 500.0, 300.0)
	RecordMetricValues("default", "test-policy", "gpu_utilization", 85.0, 80.0)
}

func TestRecordReconcileLatency(t *testing.T) {
	RecordReconcileLatency("default", "test-policy", 0.5)
}

func TestRecordReconcileError(t *testing.T) {
	RecordReconcileError("default", "test-policy", "metrics_fetch")
	RecordReconcileError("default", "test-policy", "target_not_found")
}

func TestRecordCooldownStatus(t *testing.T) {
	RecordCooldownStatus("default", "test-policy", true)
	RecordCooldownStatus("default", "test-policy", false)
}

func TestRecordLastScaleTime(t *testing.T) {
	RecordLastScaleTime("default", "test-policy", 1703123456.0)
}
