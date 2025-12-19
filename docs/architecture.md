# KubeAI Autoscaler Architecture

## Overview

KubeAI Autoscaler is a Kubernetes-native solution for dynamically scaling AI inference workloads based on real-time performance metrics.

## Architecture Diagram

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                           KubeAI Autoscaler System                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────────┐    ┌──────────────────┐    ┌────────────────────────┐│
│  │                  │    │                  │    │                        ││
│  │  CRDs            │    │  Controller      │    │  Custom Metrics        ││
│  │                  │───▶│                  │◀───│  Adapter               ││
│  │  AIInference     │    │  - Reconciler    │    │                        ││
│  │  AutoscalerPolicy│    │  - Scaler        │    │  - GPU Metrics         ││
│  │                  │    │  - Metrics Fetch │    │  - Latency Metrics     ││
│  │                  │    │                  │    │  - Queue Depth         ││
│  └──────────────────┘    └────────┬─────────┘    └───────────┬────────────┘│
│                                   │                          │             │
│                                   │                          │             │
│                                   ▼                          │             │
│                          ┌──────────────────┐                │             │
│                          │                  │                │             │
│                          │  Kubernetes API  │                │             │
│                          │                  │                │             │
│                          │  - Deployments   │                │             │
│                          │  - StatefulSets  │                │             │
│                          │  - Scale         │                │             │
│                          │                  │                │             │
│                          └──────────────────┘                │             │
│                                                              │             │
└──────────────────────────────────────────────────────────────┼─────────────┘
                                                               │
                           ┌───────────────────────────────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │                         │
              │      Prometheus         │
              │                         │
              │  - DCGM GPU Exporter    │
              │  - Inference Metrics    │
              │  - Custom Metrics       │
              │                         │
              └─────────────────────────┘
                           ▲
                           │
              ┌────────────┴────────────┐
              │                         │
              ▼                         ▼
┌─────────────────────────┐  ┌─────────────────────────┐
│                         │  │                         │
│  AI Inference Pods      │  │  NVIDIA GPU Device      │
│                         │  │  Plugin                 │
│  - LLM Server           │  │                         │
│  - Image Generation     │  │  - GPU Scheduling       │
│  - Model Serving        │  │  - Resource Allocation  │
│                         │  │                         │
└─────────────────────────┘  └─────────────────────────┘
```

## Components

### 1. Custom Resource Definitions (CRDs)

**AIInferenceAutoscalerPolicy** defines the autoscaling behavior:

| Field | Description |
|-------|-------------|
| `targetRef` | Reference to Deployment or StatefulSet |
| `minReplicas` | Minimum number of replicas |
| `maxReplicas` | Maximum number of replicas |
| `cooldownPeriod` | Time between scaling events |
| `metrics.latency` | Latency-based scaling config |
| `metrics.gpuUtilization` | GPU utilization scaling config |
| `metrics.requestQueueDepth` | Queue depth scaling config |
| `scaleUp` | Scale up behavior and policies |
| `scaleDown` | Scale down behavior and policies |

### 2. Controller / Operator

The controller continuously:

1. **Watches** AIInferenceAutoscalerPolicy CRDs
2. **Fetches** metrics from Prometheus via Custom Metrics API
3. **Calculates** desired replicas based on policy rules
4. **Scales** target Deployments/StatefulSets
5. **Updates** policy status with current state

### 3. Metrics Pipeline

```text
AI Inference Pod ──▶ Prometheus ──▶ Custom Metrics Adapter ──▶ Controller
       │
       └──▶ DCGM Exporter (GPU metrics)
```

**Supported Metrics:**

| Metric | Source | Description |
|--------|--------|-------------|
| Latency P99/P95 | Inference service | Request latency percentiles |
| GPU Utilization | DCGM Exporter | GPU compute utilization % |
| GPU Memory | DCGM Exporter | GPU memory utilization % |
| Queue Depth | Inference service | Pending requests in queue |

### 4. Autoscaling Logic

The controller uses a **multi-metric scaling algorithm**:

```text
For each enabled metric:
  1. Fetch current value from Prometheus
  2. Calculate ratio: current_value / target_value
  3. Track the maximum ratio across all metrics

Desired Replicas = ceil(current_replicas * max_ratio)
Desired Replicas = clamp(desired, min_replicas, max_replicas)

If cooldown_period has passed AND desired != current:
  Scale to desired replicas
```

### 5. Integration Points

| Integration | Purpose |
|-------------|---------|
| **Prometheus** | Metrics collection and storage |
| **KEDA** | Event-driven scaling (optional) |
| **ArgoCD** | GitOps deployment |
| **NVIDIA Device Plugin** | GPU scheduling |
| **KServe** | AI inference serving |

## Data Flow

```text
1. User applies AIInferenceAutoscalerPolicy CR
   │
   ▼
2. Controller watches and receives CR event
   │
   ▼
3. Controller fetches metrics from Prometheus
   │
   ├── GPU utilization (DCGM_FI_DEV_GPU_UTIL)
   ├── Latency P99 (histogram_quantile)
   └── Queue depth (custom metric)
   │
   ▼
4. Controller calculates desired replicas
   │
   ▼
5. Controller checks cooldown period
   │
   ▼
6. Controller updates Deployment.spec.replicas
   │
   ▼
7. Kubernetes schedules pods with GPU resources
   │
   ▼
8. Controller updates AIInferenceAutoscalerPolicy status
```

## Scaling Strategies

### Latency-Based Scaling

Scale up when P99 latency exceeds target to maintain SLA:

```yaml
metrics:
  latency:
    enabled: true
    targetP99Ms: 100  # Scale up if P99 > 100ms
```

### GPU Utilization Scaling

Scale based on GPU compute utilization:

```yaml
metrics:
  gpuUtilization:
    enabled: true
    targetPercentage: 80  # Scale up if GPU > 80%
```

### Queue Depth Scaling

Scale based on pending request queue:

```yaml
metrics:
  requestQueueDepth:
    enabled: true
    targetDepth: 10  # Scale up if queue > 10 per replica
```

## Security Considerations

- RBAC roles for controller to scale deployments
- Network policies for Prometheus access
- Secrets management for metrics authentication
- Pod security standards compliance
