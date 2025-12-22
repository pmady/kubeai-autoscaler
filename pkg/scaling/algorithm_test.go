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
	"testing"

	"github.com/stretchr/testify/assert"
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
