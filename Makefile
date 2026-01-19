# KubeAI Autoscaler Makefile

# Image URL to use all building/pushing image targets
IMG ?= kubeai-autoscaler:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

.PHONY: lint
lint: ## Run golangci-lint against code.
	golangci-lint run ./...

##@ Build

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o bin/manager ./cmd/controller/main.go

.PHONY: run
run: fmt vet ## Run a controller from your host.
	go run ./cmd/controller/main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

.PHONY: install
install: ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f crds/

.PHONY: uninstall
uninstall: ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f crds/

.PHONY: deploy
deploy: ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	kubectl apply -f deploy/

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kubectl delete -f deploy/

##@ Build Dependencies

.PHONY: generate
generate: ## Generate code (deepcopy, etc.)
	go generate ./...

.PHONY: manifests
manifests: ## Generate CRD manifests.
	controller-gen crd paths="./api/..." output:crd:artifacts:config=crds

.PHONY: clean
clean: ## Clean build artifacts.
	rm -rf bin/
	rm -f cover.out

##@ Testing

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report.
	go test ./... -coverprofile cover.out -covermode=atomic
	go tool cover -html=cover.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output.
	go test ./... -v

.PHONY: test-race
test-race: ## Run tests with race detector.
	go test ./... -race

##@ Local Development

.PHONY: run-local
run-local: ## Run controller locally with default settings.
	go run ./controller/main.go --prometheus-address=http://localhost:9090

.PHONY: kind-create
kind-create: ## Create a kind cluster for local development.
	kind create cluster --name kubeai-dev

.PHONY: kind-delete
kind-delete: ## Delete the kind cluster.
	kind delete cluster --name kubeai-dev

.PHONY: kind-load
kind-load: docker-build ## Load docker image into kind cluster.
	kind load docker-image ${IMG} --name kubeai-dev

##@ Release

.PHONY: release-build
release-build: ## Build release binaries for multiple platforms.
	GOOS=linux GOARCH=amd64 go build -o bin/manager-linux-amd64 ./controller/main.go
	GOOS=linux GOARCH=arm64 go build -o bin/manager-linux-arm64 ./controller/main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/manager-darwin-amd64 ./controller/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/manager-darwin-arm64 ./controller/main.go

.PHONY: version
version: ## Display version information.
	@echo "Version: $(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')"
	@echo "Commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Go Version: $(shell go version)"
