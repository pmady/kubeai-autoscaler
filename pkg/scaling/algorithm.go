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

package scaling

import (
	"math"
)

// Algorithm defines the interface for scaling algorithms
type Algorithm interface {
	// Calculate computes the desired replica count
	Calculate(input AlgorithmInput) int32
}

// AlgorithmInput contains the input parameters for scaling calculation
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

// Calculate implements the Algorithm interface
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

// Calculate implements the Algorithm interface
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

// Calculate implements the Algorithm interface
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
