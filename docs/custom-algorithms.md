# Custom Scaling Algorithms

This guide explains how to use and create custom scaling algorithms for KubeAI Autoscaler.

## Overview

KubeAI Autoscaler uses algorithms to determine when and how to scale your AI inference workloads. By default, it includes three built-in algorithms, and you can extend it with custom algorithms using the plugin system.

## Built-in Algorithms

### MaxRatio (Default)

The `MaxRatio` algorithm scales based on the maximum ratio across all configured metrics.

**Behavior:**

- Takes the maximum of all metric ratios (current/target)
- Scales replicas proportionally to the max ratio
- Applies tolerance to prevent unnecessary scaling

**Use Case:** Best for workloads where any single metric exceeding its target should trigger scaling.

**Example:**

```yaml
spec:
  algorithm:
    name: MaxRatio
    tolerance: 0.1 # 10% tolerance
```

### AverageRatio

The `AverageRatio` algorithm scales based on the average ratio across all metrics.

**Behavior:**

- Calculates the arithmetic mean of all metric ratios
- Scales based on the average value
- More conservative than MaxRatio

**Use Case:** Best when you want balanced consideration of all metrics.

**Example:**

```yaml
spec:
  algorithm:
    name: AverageRatio
    tolerance: 0.15
```

### WeightedRatio

The `WeightedRatio` algorithm allows you to assign different weights to each metric.

**Behavior:**

- Calculates a weighted average of metric ratios
- Metrics are weighted in order: `latencyP99`, `latencyP95`, `gpuUtilization`, `queueDepth`
- Unspecified weights default to 1.0

**Use Case:** Best when some metrics are more important than others.

**Example:**

```yaml
spec:
  algorithm:
    name: WeightedRatio
    tolerance: 0.1
    weights:
      - 2.0 # latencyP99 - 2x weight
      - 1.5 # latencyP95 - 1.5x weight
      - 1.0 # gpuUtilization - normal weight
      - 0.5 # queueDepth - half weight
```

## Configuration

### Algorithm Specification

Add the `algorithm` field to your AIInferenceAutoscalerPolicy:

```yaml
apiVersion: kubeai.io/v1alpha1
kind: AIInferenceAutoscalerPolicy
metadata:
  name: my-policy
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: inference-server
  minReplicas: 1
  maxReplicas: 10
  algorithm:
    name: MaxRatio # Algorithm name
    tolerance: 0.1 # 10% tolerance before scaling
    weights: [] # Optional: for WeightedRatio algorithm
  metrics:
    gpuUtilization:
      enabled: true
      targetPercentage: 70
```

### Parameters

| Field       | Type    | Default    | Description                         |
| ----------- | ------- | ---------- | ----------------------------------- |
| `name`      | string  | `MaxRatio` | Algorithm name (built-in or custom) |
| `tolerance` | float   | `0.1`      | Tolerance before scaling (0-1)      |
| `weights`   | []float | `[]`       | Weights for WeightedRatio algorithm |

## Custom Algorithm Plugins

### Plugin Architecture

Custom algorithms are loaded as Go plugins (`.so` files) at controller startup. The plugin system:

- Uses Go's native plugin package
- Loads plugins once at controller startup from a configurable directory
- Registers discovered plugins automatically with the algorithm registry

### Enabling Plugin Loading

Start the controller with the `--plugin-dir` flag:

```bash
kubeai-autoscaler --plugin-dir=/etc/kubeai-autoscaler/plugins
```

### Writing a Custom Algorithm

1. **Implement the ScalingAlgorithm interface:**

```go
package main

import (
    "context"
    "github.com/pmady/kubeai-autoscaler/pkg/scaling"
)

type MyAlgorithm struct {
    // Your configuration fields
}

func (a *MyAlgorithm) Name() string {
    return "MyAlgorithm"
}

func (a *MyAlgorithm) ComputeScale(ctx context.Context, input scaling.ScalingInput) (scaling.ScalingResult, error) {
    // Your scaling logic here

    return scaling.ScalingResult{
        DesiredReplicas: desiredReplicas,
        Reason:          "scaled based on my logic",
    }, nil
}

// Export the algorithm - this is required!
var Algorithm scaling.ScalingAlgorithm = &MyAlgorithm{}
```

2. **Build as a Go plugin:**

```bash
go build -buildmode=plugin -o my_algorithm.so my_algorithm.go
```

3. **Deploy the plugin:**

```bash
cp my_algorithm.so /etc/kubeai-autoscaler/plugins/
```

4. **Use in your policy:**

```yaml
spec:
  algorithm:
    name: MyAlgorithm
```

### ScalingInput Structure

Your algorithm receives the following input:

```go
type ScalingInput struct {
    CurrentReplicas int32     // Current number of replicas
    MinReplicas     int32     // Minimum replicas allowed
    MaxReplicas     int32     // Maximum replicas allowed
    MetricRatios    []float64 // Current/target ratios for each metric
    Tolerance       float64   // Configured tolerance
    PolicyName      string    // Name of the scaling policy being evaluated
    PolicyNamespace string    // Namespace of the policy (empty for cluster-scoped)
}
```

### ScalingResult Structure

Your algorithm must return:

```go
type ScalingResult struct {
    DesiredReplicas int32  // Target number of replicas
    Reason          string // Human-readable reason for the decision
}
```

### Important Considerations

1. **Thread Safety:** Your algorithm may be called concurrently from multiple goroutines.

2. **Performance:** Keep computations fast - the algorithm runs in the reconcile loop.

3. **Min/Max Constraints:** The algorithm should respect the min/max replica limits in ScalingInput.

4. **Error Handling:** Return errors only for truly exceptional cases. Return current replicas with a reason for graceful degradation.

5. **Go Version Compatibility:** Build plugins with the same Go version as the controller.

## Example: CappedSmoothRatio

See the complete example in `examples/custom-algorithm/`:

```go
// CappedSmoothRatio applies exponential smoothing and caps scaling changes
type CappedSmoothRatioAlgorithm struct {
    SmoothingFactor     float64 // Weight for new values (0-1)
    MaxScaleUpPercent   float64 // Max increase per cycle
    MaxScaleDownPercent float64 // Max decrease per cycle
}
```

This algorithm demonstrates:

- Exponential smoothing to reduce metric noise
- Capped changes to prevent aggressive scaling
- State management across reconcile cycles

## Troubleshooting

### Plugin Not Loading

| Error                                 | Cause              | Solution                                           |
| ------------------------------------- | ------------------ | -------------------------------------------------- |
| `plugin not found`                    | File doesn't exist | Check the plugin path                              |
| `plugin missing Algorithm symbol`     | Missing export     | Add `var Algorithm scaling.ScalingAlgorithm = ...` |
| `does not implement ScalingAlgorithm` | Interface mismatch | Verify `Name()` and `ComputeScale()` signatures    |
| `plugins not supported`               | Windows platform   | Use Linux or macOS                                 |

### Algorithm Not Found

If your policy specifies an unknown algorithm:

- The controller logs an error
- Falls back to MaxRatio
- Check `kubectl logs` for the controller pod

### Viewing Active Algorithms

Check the status of your policy:

```bash
kubectl get aiinferenceautoscalerpolicy my-policy -o yaml
```

The `status.lastAlgorithm` field shows which algorithm was used:

```yaml
status:
  currentReplicas: 3
  desiredReplicas: 3
  lastAlgorithm: MaxRatio
  lastScaleReason: within tolerance
```

## Platform Support

Go plugins are supported on:

- Linux (amd64, arm64)
- macOS (amd64, arm64)

**Not supported:** Windows, WebAssembly
