/*
Copyright 2023 Fr0s1.

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

package v1alpha1

import (
	"fmt"
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var configurationlog = logf.Log.WithName("configuration-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *Configuration) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-api-k8s-cloudfrosted-com-v1alpha1-configuration,mutating=true,failurePolicy=fail,sideEffects=None,groups=api.k8s.cloudfrosted.com,resources=configurations,verbs=create;update,versions=v1alpha1,name=mconfiguration.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Configuration{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Configuration) Default() {
	configurationlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
	if r.Spec.Type == "" {
		r.Spec.Type = DefaultType
	}

	if r.Spec.Setting == "" {
		r.Spec.Setting = "default setting"
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-api-k8s-cloudfrosted-com-v1alpha1-configuration,mutating=false,failurePolicy=fail,sideEffects=None,groups=api.k8s.cloudfrosted.com,resources=configurations,verbs=create;update,versions=v1alpha1,name=vconfiguration.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Configuration{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Configuration) ValidateCreate() (admission.Warnings, error) {
	configurationlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, r.validateConfiguration()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Configuration) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	configurationlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, r.validateConfiguration()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Configuration) ValidateDelete() (admission.Warnings, error) {
	configurationlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}

func (r *Configuration) validateConfiguration() error {
	var allErrs field.ErrorList

	validConfigureTypes := []ConfigurationType{DefaultType, OsType, SystemType, NetworkType}

	if !slices.Contains(validConfigureTypes, r.Spec.Type) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec").Child("type"), r.Spec.Type, fmt.Sprintf("configuration type must have one of values %v, have type %s", validConfigureTypes, r.Spec.Type)))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "api.k8s.cloudfrosted.com", Kind: "Configuration"},
		string(r.Spec.Type), allErrs)
}
