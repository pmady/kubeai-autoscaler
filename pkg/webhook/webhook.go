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

package webhook

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubeaiv1alpha1 "github.com/pmady/kubeai-autoscaler/api/v1alpha1"
)

// AIInferenceAutoscalerPolicyWebhook implements admission webhooks for AIInferenceAutoscalerPolicy
type AIInferenceAutoscalerPolicyWebhook struct{}

// SetupWebhookWithManager sets up the webhook with the manager
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&kubeaiv1alpha1.AIInferenceAutoscalerPolicy{}).
		WithValidator(&AIInferenceAutoscalerPolicyWebhook{}).
		WithDefaulter(&AIInferenceAutoscalerPolicyWebhook{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-kubeai-io-v1alpha1-aiinferenceautoscalerpolicy,mutating=true,failurePolicy=fail,sideEffects=None,groups=kubeai.io,resources=aiinferenceautoscalerpolicies,verbs=create;update,versions=v1alpha1,name=maiinferenceautoscalerpolicy.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &AIInferenceAutoscalerPolicyWebhook{}

// Default implements webhook.CustomDefaulter
func (w *AIInferenceAutoscalerPolicyWebhook) Default(ctx context.Context, obj runtime.Object) error {
	policy, ok := obj.(*kubeaiv1alpha1.AIInferenceAutoscalerPolicy)
	if !ok {
		return fmt.Errorf("expected AIInferenceAutoscalerPolicy but got %T", obj)
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Defaulting AIInferenceAutoscalerPolicy", "name", policy.Name)

	policy.SetDefaults()

	return nil
}

// +kubebuilder:webhook:path=/validate-kubeai-io-v1alpha1-aiinferenceautoscalerpolicy,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubeai.io,resources=aiinferenceautoscalerpolicies,verbs=create;update,versions=v1alpha1,name=vaiinferenceautoscalerpolicy.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &AIInferenceAutoscalerPolicyWebhook{}

// ValidateCreate implements webhook.CustomValidator
func (w *AIInferenceAutoscalerPolicyWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	policy, ok := obj.(*kubeaiv1alpha1.AIInferenceAutoscalerPolicy)
	if !ok {
		return nil, fmt.Errorf("expected AIInferenceAutoscalerPolicy but got %T", obj)
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Validating AIInferenceAutoscalerPolicy creation", "name", policy.Name)

	if err := policy.Validate(); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator
func (w *AIInferenceAutoscalerPolicyWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	policy, ok := newObj.(*kubeaiv1alpha1.AIInferenceAutoscalerPolicy)
	if !ok {
		return nil, fmt.Errorf("expected AIInferenceAutoscalerPolicy but got %T", newObj)
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Validating AIInferenceAutoscalerPolicy update", "name", policy.Name)

	if err := policy.Validate(); err != nil {
		return nil, err
	}

	// Check for immutable fields
	oldPolicy, ok := oldObj.(*kubeaiv1alpha1.AIInferenceAutoscalerPolicy)
	if !ok {
		return nil, fmt.Errorf("expected AIInferenceAutoscalerPolicy but got %T", oldObj)
	}

	if oldPolicy.Spec.TargetRef.Name != policy.Spec.TargetRef.Name {
		return admission.Warnings{"targetRef.name is being changed"}, nil
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator
func (w *AIInferenceAutoscalerPolicyWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// No validation needed for delete
	return nil, nil
}
