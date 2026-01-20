# Custom Algorithm Plugin Example

This example demonstrates how to create a custom scaling algorithm for KubeAI Autoscaler.

## CappedSmoothRatio Algorithm

The `CappedSmoothRatio` algorithm implements two key features:

1. **Exponential Smoothing**: Reduces noise in metric values by applying exponential smoothing, giving configurable weight to new vs. historical values.

2. **Capped Changes**: Limits the maximum scaling change per reconcile cycle to prevent aggressive scaling that could destabilize your workloads.

### Configuration Parameters

- `SmoothingFactor` (0.3): Weight given to new metric values (0-1). Higher values mean faster response.
- `MaxScaleUpPercent` (0.5): Maximum 50% increase per cycle
- `MaxScaleDownPercent` (0.25): Maximum 25% decrease per cycle
- `Tolerance` (0.1): 10% tolerance before scaling triggers

## Building the Plugin

```bash
# From this directory
make build

# Or manually
go build -buildmode=plugin -o capped_smooth_ratio.so smoothed_ratio.go
```

**Important**: Go plugins must be built with the same Go version and compatible build flags as the main application.

## Using the Plugin

1. Build the plugin:

   ```bash
   make build
   ```

2. Create a plugins directory and copy the plugin:

   ```bash
   mkdir -p /etc/kubeai-autoscaler/plugins
   cp capped_smooth_ratio.so /etc/kubeai-autoscaler/plugins/
   ```

3. Start the controller with the plugin directory:

   ```bash
   kubeai-autoscaler --plugin-dir=/etc/kubeai-autoscaler/plugins
   ```

4. Configure your policy to use the algorithm:
   ```yaml
   apiVersion: kubeai.io/v1alpha1
   kind: AIInferenceAutoscalerPolicy
   metadata:
     name: my-policy
   spec:
     targetRef:
       apiVersion: apps/v1
       kind: Deployment
       name: my-inference-server
     minReplicas: 1
     maxReplicas: 10
     algorithm:
       name: CappedSmoothRatio
       tolerance: 0.1
     metrics:
       gpuUtilization:
         enabled: true
         targetPercentage: 70
   ```

## Creating Your Own Algorithm

To create a custom algorithm:

1. Create a new Go file that imports `github.com/pmady/kubeai-autoscaler/pkg/scaling`

2. Implement the `ScalingAlgorithm` interface:

   ```go
   type ScalingAlgorithm interface {
       Name() string
       ComputeScale(ctx context.Context, input ScalingInput) (ScalingResult, error)
   }
   ```

3. Export your algorithm as a package-level variable named `Algorithm`:

   ```go
   var Algorithm scaling.ScalingAlgorithm = &MyAlgorithm{}
   ```

4. Build with `-buildmode=plugin`:
   ```bash
   go build -buildmode=plugin -o my_algorithm.so my_algorithm.go
   ```

## Platform Support

Go plugins are only supported on Linux and macOS. Windows is not supported.
