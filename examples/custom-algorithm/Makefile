# Makefile for building custom algorithm plugin

PLUGIN_NAME := capped_smooth_ratio
PLUGIN_FILE := $(PLUGIN_NAME).so

.PHONY: build clean test

# Build the plugin
# Note: Go plugins require the same Go version and build flags as the main application
build:
	go build -buildmode=plugin -o $(PLUGIN_FILE) smoothed_ratio.go

# Clean built artifacts
clean:
	rm -f $(PLUGIN_FILE)

# Run a simple syntax check
test:
	go build -o /dev/null smoothed_ratio.go

# Help
help:
	@echo "Available targets:"
	@echo "  build  - Build the plugin as $(PLUGIN_FILE)"
	@echo "  clean  - Remove built artifacts"
	@echo "  test   - Verify the plugin compiles"
	@echo ""
	@echo "Usage:"
	@echo "  1. Run 'make build' to create the plugin"
	@echo "  2. Copy $(PLUGIN_FILE) to your plugin directory"
	@echo "  3. Start the controller with --plugin-dir=/path/to/plugins"
