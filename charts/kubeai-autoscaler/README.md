# KubeAI Autoscaler Helm Chart

A Helm chart for deploying KubeAI Autoscaler - Kubernetes-native AI inference workload scaling.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.0+
- Prometheus (for metrics collection)

## Installation

```bash
# Add the repository (if published)
helm repo add kubeai https://pmady.github.io/kubeai-autoscaler

# Install the chart
helm install kubeai-autoscaler kubeai/kubeai-autoscaler -n kubeai-system --create-namespace
```

### Install from source

```bash
helm install kubeai-autoscaler ./charts/kubeai-autoscaler -n kubeai-system --create-namespace
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of controller replicas | `1` |
| `image.repository` | Controller image repository | `ghcr.io/pmady/kubeai-autoscaler` |
| `image.tag` | Controller image tag | `""` (uses appVersion) |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `serviceAccount.create` | Create service account | `true` |
| `prometheus.address` | Prometheus server address | `http://prometheus.monitoring.svc.cluster.local:9090` |
| `controller.leaderElection` | Enable leader election | `true` |
| `serviceMonitor.enabled` | Enable ServiceMonitor for Prometheus Operator | `false` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `64Mi` |

## Example

```bash
helm install kubeai-autoscaler ./charts/kubeai-autoscaler \
  -n kubeai-system \
  --create-namespace \
  --set prometheus.address=http://prometheus.monitoring:9090 \
  --set serviceMonitor.enabled=true
```

## Uninstallation

```bash
helm uninstall kubeai-autoscaler -n kubeai-system
```
