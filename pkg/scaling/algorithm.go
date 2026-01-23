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

// Package scaling provides scaling algorithms for the autoscaler.
package scaling

import (
	"context"
	"math"
)

// ScalingAlgorithm is the interface custom algorithms must implement
type ScalingAlgorithm interface {
	// Name returns the unique name of the algorithm
	Name() string
	// ComputeScale computes the desired replica count
	ComputeScale(ctx context.Context, input ScalingInput) (ScalingResult, error)
}

// ScalingInput contains the input parameters for scaling calculation
type ScalingInput struct {
	CurrentReplicas int32
	MinReplicas     int32
	MaxReplicas     int32
	MetricRatios    []float64 // Ratios of current/target for each metric
	Tolerance       float64
	// Policy identity for stateful algorithms to generate stable per-policy keys
	PolicyName      string
	PolicyNamespace string // Empty string for cluster-scoped policies
}

// ScalingResult contains the output of a scaling calculation
type ScalingResult struct {
	DesiredReplicas int32
	Reason          string
}

// Algorithm defines the legacy interface for scaling algorithms (deprecated)
// Use ScalingAlgorithm for new implementations
type Algorithm interface {
	// Calculate computes the desired replica count
	Calculate(input AlgorithmInput) int32
}

// AlgorithmInput contains the input parameters for scaling calculation (legacy)
type AlgorithmInput struct {
	CurrentReplicas int32
	MinReplicas     int32
	MaxReplicas     int32
	MetricRatios    []float64 // Ratios of current/target for each metric
}

// MaxRatioAlgorithm scales based on the maximum ratio across all metrics
type MaxRatioAlgorithm struct {
	// Tolerance is the percentage tolerance before scaling (e.g., 0.1 = 10%)
	Tolerance float64
}

// NewMaxRatioAlgorithm creates a new MaxRatioAlgorithm
func NewMaxRatioAlgorithm(tolerance float64) *MaxRatioAlgorithm {
	return &MaxRatioAlgorithm{
		Tolerance: tolerance,
	}
}

// Name returns the algorithm name
func (a *MaxRatioAlgorithm) Name() string {
	return "MaxRatio"
}

// ComputeScale implements the ScalingAlgorithm interface
func (a *MaxRatioAlgorithm) ComputeScale(ctx context.Context, input ScalingInput) (ScalingResult, error) {
	tolerance := input.Tolerance

	if len(input.MetricRatios) == 0 {
		desiredReplicas := input.CurrentReplicas
		// Always apply min/max constraints
		if desiredReplicas < input.MinReplicas {
			desiredReplicas = input.MinReplicas
		}
		if desiredReplicas > input.MaxReplicas {
			desiredReplicas = input.MaxReplicas
		}
		return ScalingResult{
			DesiredReplicas: desiredReplicas,
			Reason:          "no metrics available",
		}, nil
	}

	// Find the maximum ratio
	maxRatio := 1.0
	for _, ratio := range input.MetricRatios {
		if ratio > maxRatio {
			maxRatio = ratio
		}
	}

	// Apply tolerance - don't scale if within tolerance
	if maxRatio >= (1-tolerance) && maxRatio <= (1+tolerance) {
		desiredReplicas := input.CurrentReplicas
		// Always apply min/max constraints even when within tolerance
		if desiredReplicas < input.MinReplicas {
			desiredReplicas = input.MinReplicas
		}
		if desiredReplicas > input.MaxReplicas {
			desiredReplicas = input.MaxReplicas
		}
		return ScalingResult{
			DesiredReplicas: desiredReplicas,
			Reason:          "within tolerance",
		}, nil
	}

	// Calculate desired replicas
	desiredReplicas := int32(math.Ceil(float64(input.CurrentReplicas) * maxRatio))

	// Apply min/max constraints
	if desiredReplicas < input.MinReplicas {
		desiredReplicas = input.MinReplicas
	}
	if desiredReplicas > input.MaxReplicas {
		desiredReplicas = input.MaxReplicas
	}

	return ScalingResult{
		DesiredReplicas: desiredReplicas,
		Reason:          "scaled based on max ratio",
	}, nil
}

// Calculate implements the legacy Algorithm interface
func (a *MaxRatioAlgorithm) Calculate(input AlgorithmInput) int32 {
	if len(input.MetricRatios) == 0 {
		return input.CurrentReplicas
	}

	// Find the maximum ratio
	maxRatio := 1.0
	for _, ratio := range input.MetricRatios {
		if ratio > maxRatio {
			maxRatio = ratio
		}
	}

	// Apply tolerance - don't scale if within tolerance
	if maxRatio >= (1-a.Tolerance) && maxRatio <= (1+a.Tolerance) {
		return input.CurrentReplicas
	}

	// Calculate desired replicas
	desiredReplicas := int32(math.Ceil(float64(input.CurrentReplicas) * maxRatio))

	// Apply min/max constraints
	if desiredReplicas < input.MinReplicas {
		desiredReplicas = input.MinReplicas
	}
	if desiredReplicas > input.MaxReplicas {
		desiredReplicas = input.MaxReplicas
	}

	return desiredReplicas
}

// AverageRatioAlgorithm scales based on the average ratio across all metrics
type AverageRatioAlgorithm struct {
	Tolerance float64
}

// NewAverageRatioAlgorithm creates a new AverageRatioAlgorithm
func NewAverageRatioAlgorithm(tolerance float64) *AverageRatioAlgorithm {
	return &AverageRatioAlgorithm{
		Tolerance: tolerance,
	}
}

// Name returns the algorithm name
func (a *AverageRatioAlgorithm) Name() string {
	return "AverageRatio"
}

// ComputeScale implements the ScalingAlgorithm interface
func (a *AverageRatioAlgorithm) ComputeScale(ctx context.Context, input ScalingInput) (ScalingResult, error) {
	tolerance := input.Tolerance

	if len(input.MetricRatios) == 0 {
		desiredReplicas := input.CurrentReplicas
		// Always apply min/max constraints
		if desiredReplicas < input.MinReplicas {
			desiredReplicas = input.MinReplicas
		}
		if desiredReplicas > input.MaxReplicas {
			desiredReplicas = input.MaxReplicas
		}
		return ScalingResult{
			DesiredReplicas: desiredReplicas,
			Reason:          "no metrics available",
		}, nil
	}

	// Calculate average ratio
	sum := 0.0
	for _, ratio := range input.MetricRatios {
		sum += ratio
	}
	avgRatio := sum / float64(len(input.MetricRatios))

	// Apply tolerance
	if avgRatio >= (1-tolerance) && avgRatio <= (1+tolerance) {
		desiredReplicas := input.CurrentReplicas
		// Always apply min/max constraints even when within tolerance
		if desiredReplicas < input.MinReplicas {
			desiredReplicas = input.MinReplicas
		}
		if desiredReplicas > input.MaxReplicas {
			desiredReplicas = input.MaxReplicas
		}
		return ScalingResult{
			DesiredReplicas: desiredReplicas,
			Reason:          "within tolerance",
		}, nil
	}

	// Calculate desired replicas
	desiredReplicas := int32(math.Ceil(float64(input.CurrentReplicas) * avgRatio))

	// Apply min/max constraints
	if desiredReplicas < input.MinReplicas {
		desiredReplicas = input.MinReplicas
	}
	if desiredReplicas > input.MaxReplicas {
		desiredReplicas = input.MaxReplicas
	}

	return ScalingResult{
		DesiredReplicas: desiredReplicas,
		Reason:          "scaled based on average ratio",
	}, nil
}

// Calculate implements the legacy Algorithm interface
func (a *AverageRatioAlgorithm) Calculate(input AlgorithmInput) int32 {
	if len(input.MetricRatios) == 0 {
		return input.CurrentReplicas
	}

	// Calculate average ratio
	sum := 0.0
	for _, ratio := range input.MetricRatios {
		sum += ratio
	}
	avgRatio := sum / float64(len(input.MetricRatios))

	// Apply tolerance
	if avgRatio >= (1-a.Tolerance) && avgRatio <= (1+a.Tolerance) {
		return input.CurrentReplicas
	}

	// Calculate desired replicas
	desiredReplicas := int32(math.Ceil(float64(input.CurrentReplicas) * avgRatio))

	// Apply min/max constraints
	if desiredReplicas < input.MinReplicas {
		desiredReplicas = input.MinReplicas
	}
	if desiredReplicas > input.MaxReplicas {
		desiredReplicas = input.MaxReplicas
	}

	return desiredReplicas
}

// WeightedRatioAlgorithm scales based on weighted ratios
type WeightedRatioAlgorithm struct {
	Tolerance float64
	Weights   []float64
}

// NewWeightedRatioAlgorithm creates a new WeightedRatioAlgorithm
func NewWeightedRatioAlgorithm(tolerance float64, weights []float64) *WeightedRatioAlgorithm {
	return &WeightedRatioAlgorithm{
		Tolerance: tolerance,
		Weights:   weights,
	}
}

// Name returns the algorithm name
func (a *WeightedRatioAlgorithm) Name() string {
	return "WeightedRatio"
}

// SetWeights allows updating weights for the algorithm
func (a *WeightedRatioAlgorithm) SetWeights(weights []float64) {
	a.Weights = weights
}

// ComputeScale implements the ScalingAlgorithm interface
func (a *WeightedRatioAlgorithm) ComputeScale(ctx context.Context, input ScalingInput) (ScalingResult, error) {
	tolerance := input.Tolerance

	if len(input.MetricRatios) == 0 {
		desiredReplicas := input.CurrentReplicas
		// Always apply min/max constraints
		if desiredReplicas < input.MinReplicas {
			desiredReplicas = input.MinReplicas
		}
		if desiredReplicas > input.MaxReplicas {
			desiredReplicas = input.MaxReplicas
		}
		return ScalingResult{
			DesiredReplicas: desiredReplicas,
			Reason:          "no metrics available",
		}, nil
	}

	// Calculate weighted average
	weightedSum := 0.0
	totalWeight := 0.0

	for i, ratio := range input.MetricRatios {
		weight := 1.0
		if i < len(a.Weights) {
			weight = a.Weights[i]
		}
		weightedSum += ratio * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		desiredReplicas := input.CurrentReplicas
		// Always apply min/max constraints
		if desiredReplicas < input.MinReplicas {
			desiredReplicas = input.MinReplicas
		}
		if desiredReplicas > input.MaxReplicas {
			desiredReplicas = input.MaxReplicas
		}
		return ScalingResult{
			DesiredReplicas: desiredReplicas,
			Reason:          "total weight is zero",
		}, nil
	}

	weightedRatio := weightedSum / totalWeight

	// Apply tolerance
	if weightedRatio >= (1-tolerance) && weightedRatio <= (1+tolerance) {
		desiredReplicas := input.CurrentReplicas
		// Always apply min/max constraints even when within tolerance
		if desiredReplicas < input.MinReplicas {
			desiredReplicas = input.MinReplicas
		}
		if desiredReplicas > input.MaxReplicas {
			desiredReplicas = input.MaxReplicas
		}
		return ScalingResult{
			DesiredReplicas: desiredReplicas,
			Reason:          "within tolerance",
		}, nil
	}

	// Calculate desired replicas
	desiredReplicas := int32(math.Ceil(float64(input.CurrentReplicas) * weightedRatio))

	// Apply min/max constraints
	if desiredReplicas < input.MinReplicas {
		desiredReplicas = input.MinReplicas
	}
	if desiredReplicas > input.MaxReplicas {
		desiredReplicas = input.MaxReplicas
	}

	return ScalingResult{
		DesiredReplicas: desiredReplicas,
		Reason:          "scaled based on weighted ratio",
	}, nil
}

// Calculate implements the legacy Algorithm interface
func (a *WeightedRatioAlgorithm) Calculate(input AlgorithmInput) int32 {
	if len(input.MetricRatios) == 0 {
		return input.CurrentReplicas
	}

	// Calculate weighted average
	weightedSum := 0.0
	totalWeight := 0.0

	for i, ratio := range input.MetricRatios {
		weight := 1.0
		if i < len(a.Weights) {
			weight = a.Weights[i]
		}
		weightedSum += ratio * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return input.CurrentReplicas
	}

	weightedRatio := weightedSum / totalWeight

	// Apply tolerance
	if weightedRatio >= (1-a.Tolerance) && weightedRatio <= (1+a.Tolerance) {
		return input.CurrentReplicas
	}

	// Calculate desired replicas
	desiredReplicas := int32(math.Ceil(float64(input.CurrentReplicas) * weightedRatio))

	// Apply min/max constraints
	if desiredReplicas < input.MinReplicas {
		desiredReplicas = input.MinReplicas
	}
	if desiredReplicas > input.MaxReplicas {
		desiredReplicas = input.MaxReplicas
	}

	return desiredReplicas
}
