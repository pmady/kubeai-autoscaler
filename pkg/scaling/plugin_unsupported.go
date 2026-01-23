//go:build !linux && !darwin

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
	"runtime"
)

// PluginSymbolName is the symbol name that plugins must export
const PluginSymbolName = "Algorithm"

// ErrPluginsNotSupported is returned on platforms that don't support Go plugins
var ErrPluginsNotSupported = fmt.Errorf("plugins are not supported on %s/%s", runtime.GOOS, runtime.GOARCH)

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

// LoadPlugin returns an error on unsupported platforms
func LoadPlugin(path string) (ScalingAlgorithm, error) {
	return nil, ErrPluginsNotSupported
}

// LoadPlugins returns an error on unsupported platforms
func LoadPlugins(dir string) ([]ScalingAlgorithm, error) {
	return nil, ErrPluginsNotSupported
}

// LoadAndRegisterPlugins returns an error on unsupported platforms
func LoadAndRegisterPlugins(dir string, registry *Registry) error {
	return ErrPluginsNotSupported
}
