/*
Copyright 2026 KubeAI Autoscaler Authors.

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

// Package main provides a custom scaling algorithm plugin.
// This plugin implements a "CappedSmoothRatio" algorithm that:
// 1. Uses exponential smoothing to reduce noise in metric values
// 2. Caps the maximum scaling change per reconcile cycle
//
// Build this plugin with:
//
//	go build -buildmode=plugin -o capped_smooth_ratio.so smoothed_ratio.go
package main

import (
	"context"
	"math"
	"sync"

	"github.com/pmady/kubeai-autoscaler/pkg/scaling"
)

// CappedSmoothRatioAlgorithm implements a scaling algorithm with exponential
// smoothing and capped changes per cycle to prevent aggressive scaling.
type CappedSmoothRatioAlgorithm struct {
	// SmoothingFactor controls how much weight is given to new values (0-1)
	// Higher values mean faster response to changes
	SmoothingFactor float64

	// MaxScaleUpPercent is the maximum percentage increase per cycle (e.g., 0.5 = 50%)
	MaxScaleUpPercent float64

	// MaxScaleDownPercent is the maximum percentage decrease per cycle (e.g., 0.25 = 25%)
	MaxScaleDownPercent float64

	// Tolerance is the scaling tolerance
	Tolerance float64

	// mu protects the smoothed values
	mu sync.RWMutex
	// smoothedRatios stores the exponentially smoothed ratio for each policy
	smoothedRatios map[string]float64
}

// Name returns the algorithm name
func (a *CappedSmoothRatioAlgorithm) Name() string {
	return "CappedSmoothRatio"
}

// ComputeScale implements the ScalingAlgorithm interface
func (a *CappedSmoothRatioAlgorithm) ComputeScale(ctx context.Context, input scaling.ScalingInput) (scaling.ScalingResult, error) {
	tolerance := input.Tolerance
	if tolerance == 0 {
		tolerance = a.Tolerance
	}

	if len(input.MetricRatios) == 0 {
		return scaling.ScalingResult{
			DesiredReplicas: input.CurrentReplicas,
			Reason:          "no metrics available",
		}, nil
	}

	// Calculate the max ratio from current metrics
	currentMaxRatio := 1.0
	for _, ratio := range input.MetricRatios {
		if ratio > currentMaxRatio {
			currentMaxRatio = ratio
		}
	}

	// Apply exponential smoothing
	a.mu.Lock()
	if a.smoothedRatios == nil {
		a.smoothedRatios = make(map[string]float64)
	}
	// Use a key based on input parameters to track smoothing per policy
	key := policyKey(input)
	smoothedRatio, exists := a.smoothedRatios[key]
	if !exists {
		smoothedRatio = currentMaxRatio
	} else {
		// Exponential smoothing: new_value = alpha * current + (1 - alpha) * previous
		smoothedRatio = a.SmoothingFactor*currentMaxRatio + (1-a.SmoothingFactor)*smoothedRatio
	}
	a.smoothedRatios[key] = smoothedRatio
	a.mu.Unlock()

	// Check if within tolerance
	if smoothedRatio >= (1-tolerance) && smoothedRatio <= (1+tolerance) {
		return scaling.ScalingResult{
			DesiredReplicas: input.CurrentReplicas,
			Reason:          "within tolerance after smoothing",
		}, nil
	}

	// Calculate uncapped desired replicas
	uncappedDesired := float64(input.CurrentReplicas) * smoothedRatio

	// Apply scaling caps
	var desiredReplicas int32
	if smoothedRatio > 1 {
		// Scaling up - cap the increase
		maxIncrease := float64(input.CurrentReplicas) * a.MaxScaleUpPercent
		cappedDesired := math.Min(uncappedDesired, float64(input.CurrentReplicas)+maxIncrease)
		desiredReplicas = int32(math.Ceil(cappedDesired))
	} else {
		// Scaling down - cap the decrease
		maxDecrease := float64(input.CurrentReplicas) * a.MaxScaleDownPercent
		cappedDesired := math.Max(uncappedDesired, float64(input.CurrentReplicas)-maxDecrease)
		desiredReplicas = int32(math.Ceil(cappedDesired))
	}

	// Apply min/max constraints
	if desiredReplicas < input.MinReplicas {
		desiredReplicas = input.MinReplicas
	}
	if desiredReplicas > input.MaxReplicas {
		desiredReplicas = input.MaxReplicas
	}

	return scaling.ScalingResult{
		DesiredReplicas: desiredReplicas,
		Reason:          "scaled with capped smoothing",
	}, nil
}

// policyKey generates a unique key for tracking smoothed values per policy.
// Uses policy identity (namespace/name) as the primary key for stable state tracking.
func policyKey(input scaling.ScalingInput) string {
	if input.PolicyNamespace != "" {
		return input.PolicyNamespace + "/" + input.PolicyName
	}
	return input.PolicyName
}

// Algorithm is the exported symbol that the plugin loader looks for.
// It must implement the scaling.ScalingAlgorithm interface.
var Algorithm scaling.ScalingAlgorithm = &CappedSmoothRatioAlgorithm{
	SmoothingFactor:     0.3,  // 30% weight to new values
	MaxScaleUpPercent:   0.5,  // Max 50% increase per cycle
	MaxScaleDownPercent: 0.25, // Max 25% decrease per cycle
	Tolerance:           0.1,  // 10% tolerance
}
