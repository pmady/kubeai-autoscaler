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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxRatioAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		input    AlgorithmInput
		expected int32
	}{
		{
			name: "scale up when max ratio exceeds 1",
			input: AlgorithmInput{
				CurrentReplicas: 2,
				MinReplicas:     1,
				MaxReplicas:     10,
				MetricRatios:    []float64{1.5, 2.0, 1.2},
			},
			expected: 4, // 2 * 2.0 = 4
		},
		{
			name: "no scaling within tolerance",
			input: AlgorithmInput{
				CurrentReplicas: 3,
				MinReplicas:     1,
				MaxReplicas:     10,
				MetricRatios:    []float64{1.05, 0.95, 1.0},
			},
			expected: 3, // within 10% tolerance
		},
		{
			name: "respect max replicas",
			input: AlgorithmInput{
				CurrentReplicas: 5,
				MinReplicas:     1,
				MaxReplicas:     8,
				MetricRatios:    []float64{3.0},
			},
			expected: 8, // capped at max
		},
		{
			name: "respect min replicas",
			input: AlgorithmInput{
				CurrentReplicas: 1,
				MinReplicas:     2,
				MaxReplicas:     10,
				MetricRatios:    []float64{1.5}, // ratio > 1, scales to 2
			},
			expected: 2, // 1 * 1.5 = 1.5, ceil = 2, which equals min
		},
		{
			name: "empty ratios returns current",
			input: AlgorithmInput{
				CurrentReplicas: 3,
				MinReplicas:     1,
				MaxReplicas:     10,
				MetricRatios:    []float64{},
			},
			expected: 3,
		},
	}

	algo := NewMaxRatioAlgorithm(0.1)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := algo.Calculate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAverageRatioAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		input    AlgorithmInput
		expected int32
	}{
		{
			name: "scale based on average",
			input: AlgorithmInput{
				CurrentReplicas: 2,
				MinReplicas:     1,
				MaxReplicas:     10,
				MetricRatios:    []float64{2.0, 1.0, 1.5}, // avg = 1.5
			},
			expected: 3, // 2 * 1.5 = 3
		},
		{
			name: "no scaling within tolerance",
			input: AlgorithmInput{
				CurrentReplicas: 4,
				MinReplicas:     1,
				MaxReplicas:     10,
				MetricRatios:    []float64{1.05, 0.95}, // avg = 1.0
			},
			expected: 4,
		},
	}

	algo := NewAverageRatioAlgorithm(0.1)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := algo.Calculate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWeightedRatioAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		weights  []float64
		input    AlgorithmInput
		expected int32
	}{
		{
			name:    "weighted scaling",
			weights: []float64{2.0, 1.0}, // first metric has 2x weight
			input: AlgorithmInput{
				CurrentReplicas: 2,
				MinReplicas:     1,
				MaxReplicas:     10,
				MetricRatios:    []float64{2.0, 1.0}, // weighted avg = (2*2 + 1*1) / 3 = 1.67
			},
			expected: 4, // 2 * 1.67 = 3.34, ceil = 4
		},
		{
			name:    "equal weights same as average",
			weights: []float64{1.0, 1.0},
			input: AlgorithmInput{
				CurrentReplicas: 2,
				MinReplicas:     1,
				MaxReplicas:     10,
				MetricRatios:    []float64{2.0, 1.0}, // avg = 1.5
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			algo := NewWeightedRatioAlgorithm(0.1, tt.weights)
			result := algo.Calculate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Tests for the new ScalingAlgorithm interface

func TestMaxRatioAlgorithm_Name(t *testing.T) {
	algo := NewMaxRatioAlgorithm(0.1)
	assert.Equal(t, "MaxRatio", algo.Name())
}

func TestAverageRatioAlgorithm_Name(t *testing.T) {
	algo := NewAverageRatioAlgorithm(0.1)
	assert.Equal(t, "AverageRatio", algo.Name())
}

func TestWeightedRatioAlgorithm_Name(t *testing.T) {
	algo := NewWeightedRatioAlgorithm(0.1, nil)
	assert.Equal(t, "WeightedRatio", algo.Name())
}

func TestMaxRatioAlgorithm_ComputeScale(t *testing.T) {
	algo := NewMaxRatioAlgorithm(0.1)
	ctx := context.Background()

	tests := []struct {
		name           string
		input          ScalingInput
		expectedResult int32
		expectedReason string
	}{
		{
			name: "scale up based on max ratio",
			input: ScalingInput{
				CurrentReplicas: 2,
				MinReplicas:     1,
				MaxReplicas:     10,
				MetricRatios:    []float64{1.5, 2.0, 1.2},
				Tolerance:       0.1,
				PolicyName:      "test-policy",
				PolicyNamespace: "test-namespace",
			},
			expectedResult: 4,
			expectedReason: "scaled based on max ratio",
		},
		{
			name: "no metrics returns current",
			input: ScalingInput{
				CurrentReplicas: 3,
				MinReplicas:     1,
				MaxReplicas:     10,
				MetricRatios:    []float64{},
				Tolerance:       0.1,
				PolicyName:      "test-policy",
				PolicyNamespace: "test-namespace",
			},
			expectedResult: 3,
			expectedReason: "no metrics available",
		},
		{
			name: "within tolerance",
			input: ScalingInput{
				CurrentReplicas: 3,
				MinReplicas:     1,
				MaxReplicas:     10,
				MetricRatios:    []float64{1.05, 0.95},
				Tolerance:       0.1,
				PolicyName:      "test-policy",
				PolicyNamespace: "test-namespace",
			},
			expectedResult: 3,
			expectedReason: "within tolerance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := algo.ComputeScale(ctx, tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result.DesiredReplicas)
			assert.Equal(t, tt.expectedReason, result.Reason)
		})
	}
}

func TestAverageRatioAlgorithm_ComputeScale(t *testing.T) {
	algo := NewAverageRatioAlgorithm(0.1)
	ctx := context.Background()

	input := ScalingInput{
		CurrentReplicas: 2,
		MinReplicas:     1,
		MaxReplicas:     10,
		MetricRatios:    []float64{2.0, 1.0, 1.5}, // avg = 1.5
		Tolerance:       0.1,
		PolicyName:      "test-policy",
		PolicyNamespace: "test-namespace",
	}

	result, err := algo.ComputeScale(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, int32(3), result.DesiredReplicas)
	assert.Equal(t, "scaled based on average ratio", result.Reason)
}

func TestWeightedRatioAlgorithm_ComputeScale(t *testing.T) {
	algo := NewWeightedRatioAlgorithm(0.1, []float64{2.0, 1.0})
	ctx := context.Background()

	input := ScalingInput{
		CurrentReplicas: 2,
		MinReplicas:     1,
		MaxReplicas:     10,
		MetricRatios:    []float64{2.0, 1.0}, // weighted avg = (2*2 + 1*1) / 3 = 1.67
		Tolerance:       0.1,
		PolicyName:      "test-policy",
		PolicyNamespace: "test-namespace",
	}

	result, err := algo.ComputeScale(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, int32(4), result.DesiredReplicas)
	assert.Equal(t, "scaled based on weighted ratio", result.Reason)
}

func TestWeightedRatioAlgorithm_SetWeights(t *testing.T) {
	algo := NewWeightedRatioAlgorithm(0.1, nil)
	assert.Empty(t, algo.Weights)

	algo.SetWeights([]float64{1.0, 2.0, 3.0})
	assert.Equal(t, []float64{1.0, 2.0, 3.0}, algo.Weights)
}

func TestScalingAlgorithm_ToleranceFromInput(t *testing.T) {
	// Test that input tolerance overrides algorithm tolerance
	algo := NewMaxRatioAlgorithm(0.5) // High tolerance
	ctx := context.Background()

	input := ScalingInput{
		CurrentReplicas: 2,
		MinReplicas:     1,
		MaxReplicas:     10,
		MetricRatios:    []float64{1.2}, // 20% above target
		Tolerance:       0.1,            // Stricter tolerance from input
		PolicyName:      "test-policy",
		PolicyNamespace: "test-namespace",
	}

	result, err := algo.ComputeScale(ctx, input)
	require.NoError(t, err)
	// With 0.1 tolerance, 1.2 ratio should trigger scaling
	assert.Equal(t, int32(3), result.DesiredReplicas)
}

func TestScalingAlgorithm_ImplementsInterface(t *testing.T) {
	// Verify all algorithms implement ScalingAlgorithm
	var _ ScalingAlgorithm = (*MaxRatioAlgorithm)(nil)
	var _ ScalingAlgorithm = (*AverageRatioAlgorithm)(nil)
	var _ ScalingAlgorithm = (*WeightedRatioAlgorithm)(nil)
}
