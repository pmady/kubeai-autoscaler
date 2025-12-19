# KubeAI Autoscaler

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CNCF Sandbox](https://img.shields.io/badge/CNCF-Sandbox%20Proposal-326CE5?logo=cncf)](./PROPOSAL.md)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-v1.24+-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)

A Kubernetes-native solution for dynamically scaling AI inference workloads based on real-time performance metrics.

## Overview

KubeAI Autoscaler bridges the gap between AI workloads and cloud-native autoscaling by introducing AI-specific scaling logic based on:

- **GPU Utilization** - Scale based on GPU compute usage
- **Latency SLA** - Maintain response time targets (P99/P95)
- **Request Queue Depth** - Scale based on pending requests

## Why KubeAI Autoscaler?

Traditional Kubernetes autoscalers (HPA, KEDA) are CPU/memory-focused and not optimized for GPU-heavy AI inference workloads.

| Feature | HPA | KEDA | KubeAI Autoscaler |
|---------|-----|------|-------------------|
| CPU/Memory Scaling | ✅ | ✅ | ✅ |
| GPU-Aware Scaling | ❌ | ⚠️ | ✅ |
| Latency-Based Scaling | ❌ | ⚠️ | ✅ |
| AI-Specific Metrics | ❌ | ⚠️ | ✅ |
| Queue Depth Scaling | ❌ | ✅ | ✅ |

## Features

- **Custom Resource Definitions (CRDs)** - Define AI autoscaling policies declaratively
- **Prometheus Integration** - Collect GPU and latency metrics
- **Dynamic Scaling Logic** - AI-specific scaling algorithms
- **Extensible Architecture** - Support for custom metrics and scaling strategies
- **CNCF Ecosystem Integration** - Works with Prometheus, KEDA, ArgoCD

## Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                    KubeAI Autoscaler                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │   CRDs      │    │ Controller  │    │  Metrics Adapter    │  │
│  │             │───▶│             │◀───│                     │  │
│  │ AIPolicy    │    │ Reconciler  │    │ GPU/Latency/Queue   │  │
│  └─────────────┘    └──────┬──────┘    └──────────┬──────────┘  │
│                            │                      │             │
│                            ▼                      │             │
│                   ┌─────────────┐                 │             │
│                   │ Kubernetes  │                 │             │
│                   │ Deployments │                 │             │
│                   └─────────────┘                 │             │
└───────────────────────────────────────────────────┼─────────────┘
                                                    │
                                                    ▼
                                          ┌─────────────────┐
                                          │   Prometheus    │
                                          └─────────────────┘
```

See [Architecture Documentation](./docs/architecture.md) for details.

## Getting Started

### Prerequisites

- Kubernetes cluster (v1.24+)
- Prometheus installed with GPU metrics
- NVIDIA GPU device plugin (for GPU workloads)
- kubectl configured

### Installation

```bash
# Install CRDs
kubectl apply -f crds/

# Install controller
kubectl apply -f controller/
```

### Quick Start

1. **Create an AIInferenceAutoscalerPolicy:**

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
  metrics:
    latency:
      enabled: true
      targetP99Ms: 500
    gpuUtilization:
      enabled: true
      targetPercentage: 80
```

2. **Apply the policy:**

```bash
kubectl apply -f examples/basic-policy.yaml
```

3. **Check status:**

```bash
kubectl get aiap -n ai-workloads
```

## Examples

| Example | Description |
|---------|-------------|
| [basic-policy.yaml](./examples/basic-policy.yaml) | Basic autoscaling with latency and GPU metrics |
| [gpu-focused-policy.yaml](./examples/gpu-focused-policy.yaml) | GPU-intensive workload scaling |
| [latency-sla-policy.yaml](./examples/latency-sla-policy.yaml) | Strict latency SLA enforcement |

## Roadmap

### Phase 1 (MVP) - In Progress

- [x] CRD for autoscaling policy
- [ ] Basic controller logic
- [ ] Prometheus integration

### Phase 2

- [ ] Predictive scaling using AI models
- [ ] KEDA integration
- [ ] Advanced GPU scheduling

### Phase 3

- [ ] Multi-cluster support
- [ ] Service mesh integration
- [ ] Observability dashboards

## Contributing

We welcome contributions! Please see our [Contributing Guide](./CONTRIBUTING.md) for details.

## Community

- **GitHub Issues:** [Report bugs or request features](https://github.com/pmady/kubeai-autoscaler/issues)
- **Discussions:** [Ask questions and share ideas](https://github.com/pmady/kubeai-autoscaler/discussions)

## License

Apache License 2.0 - see [LICENSE](./LICENSE) for details.

## Related Projects

- [KEDA](https://keda.sh/) - Event-driven autoscaling
- [Prometheus](https://prometheus.io/) - Metrics and monitoring
- [NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator) - GPU management
- [KServe](https://kserve.github.io/) - AI inference serving
