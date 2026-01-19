# Metrics Reference

This document describes all metrics used by KubeAI Autoscaler for scaling decisions.

## GPU Metrics

### DCGM Metrics (NVIDIA Data Center GPU Manager)

| Metric | Description | Unit |
|--------|-------------|------|
| `DCGM_FI_DEV_GPU_UTIL` | GPU utilization | Percentage (0-100) |
| `DCGM_FI_DEV_MEM_COPY_UTIL` | Memory copy utilization | Percentage (0-100) |
| `DCGM_FI_DEV_FB_USED` | Framebuffer memory used | Bytes |
| `DCGM_FI_DEV_FB_FREE` | Framebuffer memory free | Bytes |
| `DCGM_FI_DEV_POWER_USAGE` | Power usage | Watts |
| `DCGM_FI_DEV_SM_CLOCK` | SM clock frequency | MHz |

### Default GPU Query

```promql
avg(DCGM_FI_DEV_GPU_UTIL)
```

### Custom GPU Queries

Filter by namespace:
```promql
avg(DCGM_FI_DEV_GPU_UTIL{namespace="ai-workloads"})
```

Filter by pod pattern:
```promql
avg(DCGM_FI_DEV_GPU_UTIL{pod=~"llm-inference.*"})
```

## Latency Metrics

### Histogram-based Latency

KubeAI Autoscaler expects latency metrics in histogram format for accurate percentile calculations.

| Metric | Description |
|--------|-------------|
| `inference_request_duration_seconds_bucket` | Request duration histogram buckets |
| `inference_request_duration_seconds_sum` | Total request duration |
| `inference_request_duration_seconds_count` | Total request count |

### Default Latency Queries

P99 Latency:
```promql
histogram_quantile(0.99, sum(rate(inference_request_duration_seconds_bucket[5m])) by (le))
```

P95 Latency:
```promql
histogram_quantile(0.95, sum(rate(inference_request_duration_seconds_bucket[5m])) by (le))
```

### NVIDIA Triton Inference Server Metrics

For Triton Inference Server, use these queries:

P99 Latency (microseconds to seconds):
```promql
histogram_quantile(0.99, 
  sum(rate(nv_inference_request_duration_us_bucket[5m])) by (le)
) / 1000000
```

### vLLM Metrics

For vLLM serving:
```promql
histogram_quantile(0.99,
  sum(rate(vllm:request_latency_seconds_bucket[5m])) by (le)
)
```

## Queue Depth Metrics

### Default Queue Query

```promql
sum(inference_request_queue_depth)
```

### NVIDIA Triton Queue Metrics

```promql
sum(nv_inference_pending_request_count)
```

### Custom Queue Metrics

Filter by service:
```promql
sum(inference_request_queue_depth{service="llm-inference"})
```

## Recording Rules

KubeAI Autoscaler provides pre-defined recording rules for efficient querying:

| Recording Rule | Description |
|----------------|-------------|
| `kubeai:gpu_utilization:avg` | Average GPU utilization by pod |
| `kubeai:inference_latency_p99:5m` | P99 latency over 5 minutes |
| `kubeai:inference_latency_p95:5m` | P95 latency over 5 minutes |
| `kubeai:request_queue_depth:sum` | Total queue depth by service |

## Metric Requirements

### For GPU-based Scaling

1. Install NVIDIA DCGM Exporter
2. Configure Prometheus to scrape DCGM metrics
3. Ensure GPU pods have proper labels for filtering

### For Latency-based Scaling

1. Instrument your inference server with Prometheus metrics
2. Use histogram metrics for accurate percentile calculations
3. Ensure consistent labeling across pods

### For Queue-based Scaling

1. Expose queue depth as a Prometheus gauge
2. Update metric on each request enqueue/dequeue
3. Label metrics with service/deployment name

## Troubleshooting

### No GPU metrics

1. Verify DCGM Exporter is running: `kubectl get pods -n gpu-operator`
2. Check Prometheus targets: `http://prometheus:9090/targets`
3. Query directly: `curl http://dcgm-exporter:9400/metrics`

### Latency metrics missing

1. Verify inference server exposes metrics
2. Check metric endpoint: `curl http://inference-server:8002/metrics`
3. Verify histogram buckets are present

### Queue depth always zero

1. Ensure queue metric is updated in real-time
2. Check if metric is a gauge (not counter)
3. Verify label selectors match your pods
