# Controller Documentation

## Overview

The KubeAI Autoscaler controller watches `AIInferenceAutoscalerPolicy` resources and automatically scales AI inference workloads based on real-time metrics.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Controller Manager                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────┐    ┌──────────────────┐               │
│  │    Reconciler    │───▶│  Metrics Client  │               │
│  │                  │    │   (Prometheus)   │               │
│  └────────┬─────────┘    └──────────────────┘               │
│           │                                                  │
│           ▼                                                  │
│  ┌──────────────────┐    ┌──────────────────┐               │
│  │ Scaling Decision │───▶│ Target Workload  │               │
│  │     Engine       │    │ (Deploy/STS)     │               │
│  └──────────────────┘    └──────────────────┘               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Reconciliation Loop

The controller follows this reconciliation flow:

1. **Fetch Policy** - Get the `AIInferenceAutoscalerPolicy` resource
2. **Get Current State** - Read current replica count from target workload
3. **Fetch Metrics** - Query Prometheus for current GPU, latency, and queue metrics
4. **Calculate Desired Replicas** - Apply scaling algorithm based on metric ratios
5. **Check Cooldown** - Ensure cooldown period has elapsed since last scale
6. **Scale Target** - Update the target Deployment or StatefulSet
7. **Update Status** - Record current metrics and replica counts

## Scaling Algorithm

The controller uses a **ratio-based scaling algorithm**:

```
scaling_ratio = max(
    current_latency / target_latency,
    current_gpu_util / target_gpu_util,
    current_queue_depth / (target_queue_depth * current_replicas)
)

desired_replicas = ceil(current_replicas * scaling_ratio)
desired_replicas = clamp(desired_replicas, min_replicas, max_replicas)
```

The highest ratio among all enabled metrics determines the scaling decision, ensuring that the most stressed metric drives the scale-up.

## Configuration

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--metrics-bind-address` | `:8080` | Address for controller metrics endpoint |
| `--health-probe-bind-address` | `:8081` | Address for health/ready probes |
| `--prometheus-address` | `http://prometheus:9090` | Prometheus server address |
| `--leader-elect` | `false` | Enable leader election for HA |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `PROMETHEUS_ADDRESS` | Override Prometheus address |
| `KUBECONFIG` | Path to kubeconfig file (for local development) |

## Metrics

The controller fetches the following metrics from Prometheus:

### Latency Metrics

- **P99 Latency**: `histogram_quantile(0.99, sum(rate(inference_request_duration_seconds_bucket[5m])) by (le))`
- **P95 Latency**: `histogram_quantile(0.95, sum(rate(inference_request_duration_seconds_bucket[5m])) by (le))`

### GPU Metrics

- **GPU Utilization**: `avg(DCGM_FI_DEV_GPU_UTIL)`

Requires NVIDIA DCGM exporter to be installed.

### Queue Metrics

- **Queue Depth**: `sum(inference_request_queue_depth)`

## Cooldown Period

The controller enforces a cooldown period between scaling events to prevent thrashing:

- Default cooldown: 5 minutes
- Configurable per-policy via `spec.cooldownPeriod` (in seconds)
- Cooldown is tracked per-policy in memory

## Supported Target Types

| Kind | API Version | Notes |
|------|-------------|-------|
| Deployment | apps/v1 | Full support |
| StatefulSet | apps/v1 | Full support |

## Example Policy

```yaml
apiVersion: kubeai.io/v1alpha1
kind: AIInferenceAutoscalerPolicy
metadata:
  name: llm-inference-policy
  namespace: ai-workloads
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-inference-server
  minReplicas: 2
  maxReplicas: 10
  cooldownPeriod: 300  # 5 minutes
  metrics:
    latency:
      enabled: true
      targetP99Ms: 500
    gpuUtilization:
      enabled: true
      targetPercentage: 80
    requestQueueDepth:
      enabled: true
      targetDepth: 10
```

## Troubleshooting

### Controller not scaling

1. Check controller logs: `kubectl logs -n kubeai-system deploy/kubeai-controller`
2. Verify Prometheus connectivity
3. Check if cooldown period has elapsed
4. Verify target workload exists

### Metrics not available

1. Ensure Prometheus is accessible from the controller
2. Verify metric names match your inference server's metrics
3. Use custom `prometheusQuery` in the policy if needed

### Status not updating

1. Check RBAC permissions for status subresource
2. Verify the policy has the correct namespace
