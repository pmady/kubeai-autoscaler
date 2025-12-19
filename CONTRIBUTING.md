# Contributing to KubeAI Autoscaler

Thank you for your interest in contributing to KubeAI Autoscaler!

## How to Contribute

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## Development Setup

### Prerequisites

- Go 1.21+
- Kubernetes cluster (kind, minikube, or remote)
- kubectl
- Docker

### Local Development

```bash
# Clone the repository
git clone https://github.com/pmady/kubeai-autoscaler.git
cd kubeai-autoscaler

# Install CRDs
kubectl apply -f crds/

# Run the controller locally
go run controller/main.go
```

### Running Tests

```bash
go test ./...
```

## Code Style

- Follow Go best practices and conventions
- Use `gofmt` for formatting
- Add comments for exported functions
- Write unit tests for new functionality

## Commit Messages

Use conventional commit format:

```
<type>(<scope>): <description>

[optional body]
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Examples:
- `feat(controller): add GPU utilization scaling`
- `fix(crd): correct validation for maxReplicas`
- `docs: update installation guide`

## Pull Request Process

1. Update documentation if needed
2. Add tests for new functionality
3. Ensure all tests pass
4. Request review from maintainers

## Reporting Issues

- Use GitHub Issues for bug reports and feature requests
- Include Kubernetes version, controller version, and logs
- Provide steps to reproduce for bugs

## Code of Conduct

Please follow our [Code of Conduct](./CODE_OF_CONDUCT.md).

## Questions?

Open a GitHub Discussion or reach out to maintainers.
