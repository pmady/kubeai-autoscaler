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
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ErrAlgorithmNotFound is returned when an algorithm is not found in the registry
type ErrAlgorithmNotFound struct {
	Name string
}

func (e ErrAlgorithmNotFound) Error() string {
	return fmt.Sprintf("algorithm not found: name=%q", e.Name)
}

// ErrAlgorithmAlreadyRegistered is returned when attempting to register a duplicate algorithm
type ErrAlgorithmAlreadyRegistered struct {
	Name string
}

func (e ErrAlgorithmAlreadyRegistered) Error() string {
	return fmt.Sprintf("algorithm already registered: name=%q", e.Name)
}

// ErrInvalidAlgorithmName is returned when an algorithm name is empty
type ErrInvalidAlgorithmName struct{}

func (e ErrInvalidAlgorithmName) Error() string {
	return "algorithm name must be non-empty"
}

// Registry manages scaling algorithms
type Registry struct {
	mu         sync.RWMutex
	algorithms map[string]ScalingAlgorithm
}

// NewRegistry creates a new algorithm registry
func NewRegistry() *Registry {
	return &Registry{
		algorithms: make(map[string]ScalingAlgorithm),
	}
}

// Register adds an algorithm to the registry
// Returns ErrAlgorithmAlreadyRegistered if an algorithm with the same name exists
func (r *Registry) Register(algorithm ScalingAlgorithm) error {
	if algorithm == nil {
		return fmt.Errorf("cannot register nil algorithm")
	}

	name := strings.TrimSpace(algorithm.Name())
	if name == "" {
		return ErrInvalidAlgorithmName{}
	}
	if _, exists := r.algorithms[name]; exists {
		return ErrAlgorithmAlreadyRegistered{Name: name}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.algorithms[name] = algorithm
	return nil
}

// MustRegister adds an algorithm to the registry and panics on error
func (r *Registry) MustRegister(algorithm ScalingAlgorithm) {
	if err := r.Register(algorithm); err != nil {
		panic(err)
	}
}

// Get retrieves an algorithm by name
// Returns ErrAlgorithmNotFound if the algorithm doesn't exist
func (r *Registry) Get(name string) (ScalingAlgorithm, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	algorithm, exists := r.algorithms[name]
	if !exists {
		return nil, ErrAlgorithmNotFound{Name: name}
	}

	return algorithm, nil
}

// List returns all registered algorithm names sorted alphabetically
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.algorithms))
	for name := range r.algorithms {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Has checks if an algorithm with the given name exists
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.algorithms[name]
	return exists
}

// DefaultRegistry is the global algorithm registry
var DefaultRegistry = NewRegistry()

// DefaultTolerance is the default tolerance for built-in algorithms
const DefaultTolerance = 0.1

func init() {
	// Register built-in algorithms with the default registry
	DefaultRegistry.MustRegister(NewMaxRatioAlgorithm(DefaultTolerance))
	DefaultRegistry.MustRegister(NewAverageRatioAlgorithm(DefaultTolerance))
	DefaultRegistry.MustRegister(NewWeightedRatioAlgorithm(DefaultTolerance, nil))
}

// Register adds an algorithm to the default registry
func Register(algorithm ScalingAlgorithm) error {
	return DefaultRegistry.Register(algorithm)
}

// Get retrieves an algorithm from the default registry
func Get(name string) (ScalingAlgorithm, error) {
	return DefaultRegistry.Get(name)
}

// List returns all algorithm names from the default registry
func List() []string {
	return DefaultRegistry.List()
}
