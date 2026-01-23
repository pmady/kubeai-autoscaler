/*
Copyright 2025 KubeAI Autoscaler Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package main is the entry point for the kubeai-autoscaler controller.
package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kubeaiv1alpha1 "github.com/pmady/kubeai-autoscaler/api/v1alpha1"
	"github.com/pmady/kubeai-autoscaler/pkg/controller"
	"github.com/pmady/kubeai-autoscaler/pkg/metrics"
	"github.com/pmady/kubeai-autoscaler/pkg/scaling"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kubeaiv1alpha1.AddToScheme(scheme))
}

// stringListDiff returns elements in 'after' that are not in 'before' (set difference).
func stringListDiff(before, after []string) []string {
	beforeSet := make(map[string]struct{}, len(before))
	for _, s := range before {
		beforeSet[s] = struct{}{}
	}

	var diff []string
	for _, s := range after {
		if _, exists := beforeSet[s]; !exists {
			diff = append(diff, s)
		}
	}
	return diff
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var prometheusAddr string
	var pluginDir string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&prometheusAddr, "prometheus-address", "http://prometheus:9090", "The address of the Prometheus server.")
	flag.StringVar(&pluginDir, "plugin-dir", "", "Directory containing custom algorithm plugins (.so files)")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "kubeai-autoscaler.kubeai.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Load custom algorithm plugins
	if pluginDir != "" {
		setupLog.Info("loading custom algorithm plugins", "directory", pluginDir)
		algorithmsBefore := scaling.List()
		if err := scaling.LoadAndRegisterPlugins(pluginDir, scaling.DefaultRegistry); err != nil {
			setupLog.Error(err, "failed to load some plugins, continuing with available algorithms")
		}
		algorithmsAfter := scaling.List()
		addedByPlugins := stringListDiff(algorithmsBefore, algorithmsAfter)
		setupLog.Info("algorithms added by plugins", "algorithms", addedByPlugins)
		setupLog.Info("registered algorithms", "algorithms", algorithmsAfter)
	}

	// Create Prometheus metrics client
	var metricsClient metrics.Client
	if prometheusAddr != "" {
		metricsClient, err = metrics.NewPrometheusClient(prometheusAddr)
		if err != nil {
			setupLog.Error(err, "unable to create Prometheus client, continuing without metrics")
		}
	}

	// Setup reconciler
	reconciler := controller.NewReconciler(mgr.GetClient(), mgr.GetScheme(), metricsClient, scaling.DefaultRegistry)
	if err = reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AIInferenceAutoscalerPolicy")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
