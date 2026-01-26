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

package scaling

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAlgorithm is a simple mock implementation for testing
type mockAlgorithm struct {
	name string
}

func (m *mockAlgorithm) Name() string {
	return m.name
}

func (m *mockAlgorithm) ComputeScale(_ context.Context, input ScalingInput) (ScalingResult, error) {
	return ScalingResult{
		DesiredReplicas: input.CurrentReplicas,
		Reason:          "mock",
	}, nil
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name      string
		algorithm ScalingAlgorithm
		wantErr   bool
		errType   interface{}
	}{
		{
			name:      "register valid algorithm",
			algorithm: &mockAlgorithm{name: "TestAlgo"},
			wantErr:   false,
		},
		{
			name:      "register nil algorithm",
			algorithm: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			err := r.Register(tt.algorithm)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegistry_DuplicateRegistration(t *testing.T) {
	r := NewRegistry()

	algo1 := &mockAlgorithm{name: "DupeAlgo"}
	algo2 := &mockAlgorithm{name: "DupeAlgo"}

	// First registration should succeed
	err := r.Register(algo1)
	require.NoError(t, err)

	// Second registration with same name should fail
	err = r.Register(algo2)
	require.Error(t, err)

	var duplicateErr ErrAlgorithmAlreadyRegistered
	assert.ErrorAs(t, err, &duplicateErr)
	assert.Equal(t, "DupeAlgo", duplicateErr.Name)
}

func TestRegistry_InvalidAlgorithmName(t *testing.T) {
	r := NewRegistry()

	algo := &mockAlgorithm{name: "   "}

	err := r.Register(algo)
	require.Error(t, err)

	var invalidNameErr ErrInvalidAlgorithmName
	assert.ErrorAs(t, err, &invalidNameErr)
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	algo := &mockAlgorithm{name: "GetAlgo"}
	require.NoError(t, r.Register(algo))

	tests := []struct {
		name     string
		algoName string
		wantErr  bool
	}{
		{
			name:     "get existing algorithm",
			algoName: "GetAlgo",
			wantErr:  false,
		},
		{
			name:     "get non-existent algorithm",
			algoName: "NonExistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Get(tt.algoName)
			if tt.wantErr {
				assert.Error(t, err)
				var notFoundErr ErrAlgorithmNotFound
				assert.ErrorAs(t, err, &notFoundErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.algoName, result.Name())
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	// Register some algorithms
	require.NoError(t, r.Register(&mockAlgorithm{name: "Zebra"}))
	require.NoError(t, r.Register(&mockAlgorithm{name: "Alpha"}))
	require.NoError(t, r.Register(&mockAlgorithm{name: "Mango"}))

	list := r.List()

	// Should be sorted alphabetically
	assert.Equal(t, []string{"Alpha", "Mango", "Zebra"}, list)
}

func TestRegistry_Has(t *testing.T) {
	r := NewRegistry()
	require.NoError(t, r.Register(&mockAlgorithm{name: "HasAlgo"}))

	assert.True(t, r.Has("HasAlgo"))
	assert.False(t, r.Has("NonExistent"))
}

func TestRegistry_MustRegister(t *testing.T) {
	r := NewRegistry()
	algo := &mockAlgorithm{name: "MustAlgo"}

	// Should not panic
	assert.NotPanics(t, func() {
		r.MustRegister(algo)
	})

	// Duplicate registration should panic
	assert.Panics(t, func() {
		r.MustRegister(algo)
	})
}

func TestRegistry_ThreadSafety(t *testing.T) {
	r := NewRegistry()

	// Pre-register some algorithms
	for i := 0; i < 10; i++ {
		require.NoError(t, r.Register(&mockAlgorithm{name: "Algo" + string(rune('A'+i))}))
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = r.Get("AlgoA")
				_ = r.List()
				_ = r.Has("AlgoB")
			}
		}()
	}

	wg.Wait()
}

func TestDefaultRegistry_BuiltInAlgorithms(t *testing.T) {
	// Test that built-in algorithms are registered
	algorithms := []string{"MaxRatio", "AverageRatio", "WeightedRatio"}

	for _, name := range algorithms {
		t.Run(name, func(t *testing.T) {
			algo, err := DefaultRegistry.Get(name)
			require.NoError(t, err)
			assert.Equal(t, name, algo.Name())
		})
	}
}

func TestPackageFunctions(t *testing.T) {
	// Test package-level convenience functions
	list := List()
	assert.Contains(t, list, "MaxRatio")

	algo, err := Get("MaxRatio")
	require.NoError(t, err)
	assert.Equal(t, "MaxRatio", algo.Name())
}
