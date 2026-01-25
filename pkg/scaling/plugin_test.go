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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadPlugin_FileNotFound(t *testing.T) {
	_, err := LoadPlugin("/nonexistent/path/plugin.so")
	assert.Error(t, err)

	var notFoundErr ErrPluginNotFound
	assert.ErrorAs(t, err, &notFoundErr)
	assert.Equal(t, "/nonexistent/path/plugin.so", notFoundErr.Path)
}

func TestLoadPlugin_InvalidFile(t *testing.T) {
	// Create a temporary file that's not a valid plugin
	tmpDir := t.TempDir()
	invalidPlugin := filepath.Join(tmpDir, "invalid.so")

	err := os.WriteFile(invalidPlugin, []byte("not a plugin"), 0600) // #nosec G306
	assert.NoError(t, err)

	_, err = LoadPlugin(invalidPlugin)
	assert.Error(t, err)

	var loadErr ErrPluginLoadFailed
	assert.ErrorAs(t, err, &loadErr)
	assert.Equal(t, invalidPlugin, loadErr.Path)
}

func TestLoadPlugins_DirectoryNotFound(t *testing.T) {
	_, err := LoadPlugins("/nonexistent/directory")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLoadPlugins_NotADirectory(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "notadir")

	err := os.WriteFile(tmpFile, []byte("file"), 0600) // #nosec G306
	assert.NoError(t, err)

	_, err = LoadPlugins(tmpFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestLoadPlugins_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	algorithms, err := LoadPlugins(tmpDir)
	assert.NoError(t, err)
	assert.Empty(t, algorithms)
}

func TestLoadAndRegisterPlugins_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry()

	err := LoadAndRegisterPlugins(tmpDir, registry)
	assert.NoError(t, err)
	assert.Empty(t, registry.List())
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrPluginNotFound",
			err:      ErrPluginNotFound{Path: "/path/to/plugin.so"},
			expected: `plugin not found: path="/path/to/plugin.so"`,
		},
		{
			name:     "ErrPluginSymbolNotFound",
			err:      ErrPluginSymbolNotFound{Path: "/path/to/plugin.so"},
			expected: `plugin missing Algorithm symbol: path="/path/to/plugin.so"`,
		},
		{
			name:     "ErrPluginInterfaceMismatch",
			err:      ErrPluginInterfaceMismatch{Path: "/path/to/plugin.so"},
			expected: `plugin Algorithm does not implement ScalingAlgorithm: path="/path/to/plugin.so"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}
