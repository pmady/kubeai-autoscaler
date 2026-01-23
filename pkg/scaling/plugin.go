//go:build linux || darwin

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
	"os"
	"path/filepath"
	"plugin"
)

// PluginSymbolName is the symbol name that plugins must export
const PluginSymbolName = "Algorithm"

// ErrPluginNotFound is returned when a plugin file cannot be found
type ErrPluginNotFound struct {
	Path string
}

func (e ErrPluginNotFound) Error() string {
	return fmt.Sprintf("plugin not found: path=%q", e.Path)
}

// ErrPluginLoadFailed is returned when a plugin fails to load
type ErrPluginLoadFailed struct {
	Path  string
	Cause error
}

func (e ErrPluginLoadFailed) Error() string {
	return fmt.Sprintf("failed to load plugin: path=%q, error=%q", e.Path, e.Cause)
}

// ErrPluginSymbolNotFound is returned when the Algorithm symbol is not found
type ErrPluginSymbolNotFound struct {
	Path string
}

func (e ErrPluginSymbolNotFound) Error() string {
	return fmt.Sprintf("plugin missing %s symbol: path=%q", PluginSymbolName, e.Path)
}

// ErrPluginInterfaceMismatch is returned when the symbol doesn't implement ScalingAlgorithm
type ErrPluginInterfaceMismatch struct {
	Path string
}

func (e ErrPluginInterfaceMismatch) Error() string {
	return fmt.Sprintf("plugin %s does not implement ScalingAlgorithm: path=%q", PluginSymbolName, e.Path)
}

// LoadPlugin loads a single plugin from the given path
// The plugin must export a symbol named "Algorithm" that implements ScalingAlgorithm
func LoadPlugin(path string) (ScalingAlgorithm, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, ErrPluginNotFound{Path: path}
	}

	// Open the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return nil, ErrPluginLoadFailed{Path: path, Cause: err}
	}

	// Look up the Algorithm symbol
	sym, err := p.Lookup(PluginSymbolName)
	if err != nil {
		return nil, ErrPluginSymbolNotFound{Path: path}
	}

	// Assert that the symbol implements ScalingAlgorithm
	// The plugin should export a pointer to a ScalingAlgorithm implementation
	algorithm, ok := sym.(ScalingAlgorithm)
	if !ok {
		// Try pointer to ScalingAlgorithm
		algorithmPtr, ok := sym.(*ScalingAlgorithm)
		if !ok {
			return nil, ErrPluginInterfaceMismatch{Path: path}
		}
		algorithm = *algorithmPtr
	}

	return algorithm, nil
}

// LoadPlugins loads all plugins from the given directory
// Returns a slice of loaded algorithms and any errors encountered
func LoadPlugins(dir string) ([]ScalingAlgorithm, error) {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin directory not found: path=%q", dir)
		}
		return nil, fmt.Errorf("failed to stat plugin directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("plugin path is not a directory: path=%q", dir)
	}

	// Find all .so files in the directory
	pattern := filepath.Join(dir, "*.so")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob plugins: %w", err)
	}

	var algorithms []ScalingAlgorithm
	var loadErrors []error

	for _, path := range matches {
		algorithm, err := LoadPlugin(path)
		if err != nil {
			loadErrors = append(loadErrors, err)
			continue
		}
		algorithms = append(algorithms, algorithm)
	}

	// Return combined error if any plugins failed to load
	if len(loadErrors) > 0 {
		return algorithms, fmt.Errorf("failed to load %d plugin(s): %v", len(loadErrors), loadErrors)
	}

	return algorithms, nil
}

// LoadAndRegisterPlugins loads all plugins from the directory and registers them
func LoadAndRegisterPlugins(dir string, registry *Registry) error {
	algorithms, err := LoadPlugins(dir)
	if err != nil {
		// Log but don't fail if some plugins couldn't be loaded
		// The successfully loaded plugins will still be registered
		if len(algorithms) == 0 {
			return err
		}
	}

	var registrationErrors []error
	for _, alg := range algorithms {
		if err := registry.Register(alg); err != nil {
			registrationErrors = append(registrationErrors, err)
		}
	}

	if len(registrationErrors) > 0 {
		return fmt.Errorf("failed to register %d algorithm(s): %v", len(registrationErrors), registrationErrors)
	}

	return nil
}
