# CNCF Sandbox Proposal: KubeAI Autoscaler

## Project Name

**KubeAI Autoscaler**

## Summary

KubeAI Autoscaler is a Kubernetes-native solution for dynamically scaling AI inference workloads based on real-time performance metrics such as latency, GPU utilization, and request throughput. Unlike traditional autoscalers that rely on CPU/memory metrics, this project introduces AI-specific scaling logic to optimize resource usage and improve inference performance in cloud-native environments.

## Problem Statement

Current Kubernetes autoscaling solutions (Horizontal Pod Autoscaler, KEDA) are designed for general workloads and do not account for GPU-intensive AI inference jobs. AI workloads often require:

- **GPU-aware scaling** for cost efficiency
- **Latency-based scaling** to maintain SLA for inference
- **Custom metrics** like model response time and queue depth

Without these capabilities, organizations face:

- Over-provisioning of expensive GPU resources
- Poor inference performance under variable load
- Lack of observability for AI-specific metrics

## Goals

1. Provide a Kubernetes controller that scales AI inference pods based on custom metrics
2. Integrate with Prometheus for metric collection and KEDA for event-driven scaling
3. Support GPU-aware scheduling using Kubernetes device plugins
4. Offer CRDs for defining AI autoscaling policies (e.g., latency thresholds, GPU utilization targets)

## Key Features

- **Custom Metrics Adapter** for AI workloads (latency, GPU usage, request queue depth)
- **Dynamic Scaling Logic** based on AI-specific SLAs
- **Integration with CNCF Ecosystem:**
  - Prometheus for metrics
  - KEDA for event-driven scaling
  - ArgoCD for GitOps-based deployment
- **Extensible Architecture** for future ML pipeline integration

## Roadmap

### Phase 1 (MVP)

- [ ] Implement CRD for autoscaling policy (`AIInferenceAutoscalerPolicy`)
- [ ] Basic controller logic for scaling based on latency and GPU utilization
- [ ] Prometheus integration for metrics collection
- [ ] Collect GPU and latency metrics via Prometheus
- [ ] Basic scaling logic for inference pods

### Phase 2

- [ ] Add predictive scaling using AI models
- [ ] Integration with KEDA for event-driven scaling
- [ ] Advanced GPU-aware scheduling optimizations

### Phase 3

- [ ] Multi-cluster support
- [ ] Service mesh integration for secure metric collection
- [ ] Observability dashboards for AI workloads

## Community & Governance

- **License:** Apache 2.0
- **Governance Model:** Maintainers + community contributors
- **Initial Contributors:** [pavan4devops@gmail.com](mailto:pavan4devops@gmail.com)
- Open for CNCF community participation

## Why CNCF Sandbox?

1. **Aligns with CNCF's mission** to advance cloud-native technologies
2. **Bridges the gap** between AI workloads and Kubernetes-native scaling
3. **Early-stage** but addresses a growing need in AI/ML infrastructure

## Repository & Documentation

- **GitHub:** [github.com/pmady/kubeai-autoscaler](https://github.com/pmady/kubeai-autoscaler)
- **Documentation:** Installation guide, CRD specs, examples

## Architecture

### Components

#### 1. Custom Resource Definitions (CRDs)

**AIInferenceAutoscalerPolicy** - Defines scaling rules:
- Latency threshold
- GPU utilization target
- Min/max replicas

#### 2. Controller / Operator

- Watches CRDs and metrics
- Applies scaling decisions to Kubernetes Deployment or StatefulSet

#### 3. Metrics Pipeline

- **Prometheus** scrapes:
  - GPU metrics (via NVIDIA DCGM exporter or similar)
  - Latency metrics (from inference service)
- **Custom Metrics Adapter** exposes these metrics to Kubernetes API

#### 4. Autoscaling Logic

Calculates desired replicas based on:
- Latency SLA
- GPU utilization
- Request queue depth

#### 5. Integration Points

- **KEDA** for event-driven scaling (optional)
- **ArgoCD** for GitOps deployment
- **Device Plugin** for GPU scheduling

### Flow

```text
[User] --> [CRD: AIInferenceAutoscalerPolicy] --> [Controller]
                                                       |
                                                       v
                                          [Kubernetes API: Scale Deployment]
                                                       ^
                                                       |
[Prometheus] --> [Custom Metrics Adapter] --> [Controller]

[GPU Device Plugin] --> [Kubernetes Scheduler]
```

### Detailed Architecture Diagram

```text
┌─────────────────────────────────────────────────────────────────┐
│                    KubeAI Autoscaler                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │   CRDs      │    │ Controller  │    │  Metrics Adapter    │  │
│  │             │───▶│             │◀───│                     │  │
│  │ AIScaler    │    │ Reconciler  │    │ GPU/Latency/Queue   │  │
│  │ AIPolicy    │    │             │    │                     │  │
│  └─────────────┘    └──────┬──────┘    └──────────┬──────────┘  │
│                            │                      │             │
│                            ▼                      │             │
│                   ┌─────────────┐                 │             │
│                   │ Kubernetes  │                 │             │
│                   │ Deployments │                 │             │
│                   └─────────────┘                 │             │
│                                                   │             │
└───────────────────────────────────────────────────┼─────────────┘
                                                    │
                    ┌───────────────────────────────┘
                    │
                    ▼
          ┌─────────────────┐
          │   Prometheus    │
          │   (Metrics)     │
          └─────────────────┘
```

## Related Projects

| Project | Relationship |
|---------|--------------|
| [KEDA](https://keda.sh/) | Event-driven scaling integration |
| [Prometheus](https://prometheus.io/) | Metrics collection |
| [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) | GPU device plugin |
| [KServe](https://kserve.github.io/) | AI inference serving |

## Contact

- **Maintainer:** Pavan Madduri
- **Email:** pavan4devops@gmail.com
- **GitHub:** [@pmady](https://github.com/pmady)
