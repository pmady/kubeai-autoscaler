# Getting Started with KubeAI Autoscaler

This guide will help you get started with KubeAI Autoscaler for scaling your AI inference workloads.

## Prerequisites

- Kubernetes cluster (v1.24+)
- kubectl configured to access your cluster
- Prometheus installed (for metrics collection)
- NVIDIA GPU device plugin (for GPU workloads)

## Installation

### 1. Install CRDs

```bash
kubectl apply -f https://raw.githubusercontent.com/pmady/kubeai-autoscaler/main/crds/aiinferenceautoscalerpolicy.yaml
```

### 2. Install the Controller

```bash
kubectl apply -f https://raw.githubusercontent.com/pmady/kubeai-autoscaler/main/deploy/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/pmady/kubeai-autoscaler/main/deploy/rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/pmady/kubeai-autoscaler/main/deploy/deployment.yaml
```

### 3. Verify Installation

```bash
kubectl get pods -n kubeai-system
kubectl get crd aiinferenceautoscalerpolicies.kubeai.io
```

## Creating Your First Policy

### Basic Latency-Based Scaling

Create a file named `my-policy.yaml`:

```yaml
apiVersion: kubeai.io/v1alpha1
kind: AIInferenceAutoscalerPolicy
metadata:
  name: my-inference-policy
  namespace: default
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-inference-server
  minReplicas: 2
  maxReplicas: 10
  cooldownPeriod: 300
  metrics:
    latency:
      enabled: true
      targetP99Ms: 500
```

Apply the policy:

```bash
kubectl apply -f my-policy.yaml
```

### GPU-Based Scaling

```yaml
apiVersion: kubeai.io/v1alpha1
kind: AIInferenceAutoscalerPolicy
metadata:
  name: gpu-scaling-policy
  namespace: default
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: gpu-inference-server
  minReplicas: 1
  maxReplicas: 8
  metrics:
    gpuUtilization:
      enabled: true
      targetPercentage: 80
```

### Multi-Metric Scaling

```yaml
apiVersion: kubeai.io/v1alpha1
kind: AIInferenceAutoscalerPolicy
metadata:
  name: multi-metric-policy
  namespace: default
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-server
  minReplicas: 2
  maxReplicas: 20
  cooldownPeriod: 180
  metrics:
    latency:
      enabled: true
      targetP99Ms: 1000
    gpuUtilization:
      enabled: true
      targetPercentage: 75
    requestQueueDepth:
      enabled: true
      targetDepth: 10
```

## Monitoring

### Check Policy Status

```bash
kubectl get aiap -A
kubectl describe aiap my-inference-policy
```

### View Events

```bash
kubectl get events --field-selector involvedObject.kind=AIInferenceAutoscalerPolicy
```

## Custom Prometheus Queries

You can specify custom Prometheus queries for each metric:

```yaml
spec:
  metrics:
    latency:
      enabled: true
      targetP99Ms: 500
      prometheusQuery: 'histogram_quantile(0.99, sum(rate(my_custom_latency_bucket[5m])) by (le))'
    gpuUtilization:
      enabled: true
      targetPercentage: 80
      prometheusQuery: 'avg(my_custom_gpu_metric{pod=~"my-inference.*"})'
```

## Troubleshooting

### Policy Not Scaling

1. Check if the target deployment exists:

   ```bash
   kubectl get deployment my-inference-server
   ```

2. Check controller logs:

   ```bash
   kubectl logs -n kubeai-system -l app.kubernetes.io/name=kubeai-autoscaler
   ```

3. Verify Prometheus is accessible:

   ```bash
   kubectl port-forward -n monitoring svc/prometheus 9090:9090
   ```

### Metrics Not Available

Ensure your inference service exposes the required metrics:

- For latency: `inference_request_duration_seconds` histogram
- For GPU: DCGM exporter metrics (`DCGM_FI_DEV_GPU_UTIL`)
- For queue depth: `inference_request_queue_depth` gauge

## Next Steps

- Read the [Architecture Documentation](./architecture.md)
- Explore [Example Policies](../examples/)
- Join our [Community Discussions](https://github.com/pmady/kubeai-autoscaler/discussions)
