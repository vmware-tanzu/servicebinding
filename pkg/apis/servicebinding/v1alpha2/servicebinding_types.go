/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha2

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/tracker"

	labsinternalv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labsinternal/v1alpha1"
)

const (
	ServiceBindingAnnotationKey = GroupName + "/service-binding"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceBindingSpec   `json:"spec,omitempty"`
	Status ServiceBindingStatus `json:"status,omitempty"`
}

var (
	// Check that ServiceBinding can be validated and defaulted.
	_ apis.Validatable   = (*ServiceBinding)(nil)
	_ apis.Defaultable   = (*ServiceBinding)(nil)
	_ kmeta.OwnerRefable = (*ServiceBinding)(nil)
	_ duckv1.KRShaped    = (*ServiceBinding)(nil)
)

type ServiceBindingSpec struct {
	// Name of the service binding on disk, defaults to this resource's name
	Name string `json:"name,omitempty"`
	// Type of the provisioned service. The value is exposed directly as the
	// `type` in the mounted binding
	// +optional
	Type string `json:"type,omitempty"`
	// Provider of the provisioned service. The value is exposed directly as the
	// `provider` in the mounted binding
	// +optional
	Provider string `json:"provider,omitempty"`

	// Application resource to inject the binding into
	Application *ApplicationReference `json:"application,omitempty"`
	// Service referencing the binding secret
	Service *tracker.Reference `json:"service,omitempty"`

	// Env projects keys from the binding secret into the application as
	// environment variables
	Env []EnvVar `json:"env,omitempty"`
}

type ApplicationReference = labsinternalv1alpha1.ApplicationReference

type EnvVar = labsinternalv1alpha1.EnvVar

type ServiceBindingStatus struct {
	// ObservedGeneration is the 'Generation' of the ServiceBinding that
	// was last processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions the latest available observations of a ServiceBinding's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Binding is a reference to the Secret being bound.
	// +optional
	Binding *corev1.LocalObjectReference `json:"binding,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ServiceBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBinding `json:"items"`
}

func (b *ServiceBinding) Validate(ctx context.Context) (errs *apis.FieldError) {
	if b.Spec.Application == nil {
		errs = errs.Also(
			apis.ErrMissingField("spec.application"),
		)
	} else {
		// tracker.Reference requires a Namespace
		a := b.Spec.Application.DeepCopy()
		a.Namespace = "fake"
		errs = errs.Also(
			a.Validate(ctx).ViaField("spec.application"),
		)
		if b.Spec.Application.Namespace != "" {
			errs = errs.Also(
				apis.ErrDisallowedFields("spec.application.namespace"),
			)
		}
	}

	if b.Spec.Service == nil {
		errs = errs.Also(
			apis.ErrMissingField("spec.service"),
		)
	} else {
		// tracker.Reference requires a Namespace
		s := b.Spec.Service.DeepCopy()
		s.Namespace = "fake"
		errs = errs.Also(
			s.Validate(ctx).ViaField("spec.service"),
		)
		if b.Spec.Service.Namespace != "" {
			errs = errs.Also(
				apis.ErrDisallowedFields("spec.service.namespace"),
			)
		}
		if b.Spec.Service.Name == "" {
			errs = errs.Also(
				apis.ErrMissingField("spec.service.name"),
			)
		}
	}

	envSet := map[string][]int{}
	for i, e := range b.Spec.Env {
		errs = errs.Also(
			e.Validate(ctx).ViaFieldIndex("env", i).ViaField("spec"),
		)
		if _, ok := envSet[e.Name]; !ok {
			envSet[e.Name] = []int{}
		}
		envSet[e.Name] = append(envSet[e.Name], i)
	}
	// look for conflicting names
	for _, v := range envSet {
		if len(v) != 1 {
			paths := make([]string, len(v))
			for pi, i := range v {
				paths[i] = fmt.Sprintf("spec.env[%d].name", pi)
			}
			errs = errs.Also(
				apis.ErrMultipleOneOf(paths...),
			)
		}
	}

	return errs
}

func (b *ServiceBinding) SetDefaults(context.Context) {
	if b.Spec.Name == "" {
		b.Spec.Name = b.Name
	}
}

func (b *ServiceBinding) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ServiceBinding")
}
