/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha2

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/tracker"

	serviceinternalv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/serviceinternal/v1alpha2"
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

	// Mappings create new binding secret keys from literal values or templated
	// from existing keys
	Mappings []Mapping `json:"mappings,omitempty"`
}

type ApplicationReference = serviceinternalv1alpha2.ApplicationReference

type EnvVar = serviceinternalv1alpha2.EnvVar

type Mapping struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ServiceBindingStatus struct {
	duckv1.Status `json:",inline"`
	Binding       *corev1.LocalObjectReference `json:"binding,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ServiceBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBinding `json:"items"`
}

func (b *ServiceBinding) Validate(ctx context.Context) (errs *apis.FieldError) {
	errs = errs.Also(
		b.Spec.Application.Validate(ctx).ViaField("spec.application"),
	)
	if b.Spec.Application.Namespace != b.Namespace {
		errs = errs.Also(
			apis.ErrDisallowedFields("spec.application.namespace"),
		)
	}
	errs = errs.Also(
		b.Spec.Service.Validate(ctx).ViaField("spec.service"),
	)
	if b.Spec.Service.Namespace != b.Namespace {
		errs = errs.Also(
			apis.ErrDisallowedFields("spec.service.namespace"),
		)
	}
	if b.Spec.Service.Name == "" {
		errs = errs.Also(
			apis.ErrMissingField("spec.service.name"),
		)
	}
	for i, e := range b.Spec.Env {
		errs = errs.Also(
			e.Validate(ctx).ViaFieldIndex("env", i).ViaField("spec"),
		)
		// TODO look for conflicting names
	}
	for i, m := range b.Spec.Mappings {
		errs = errs.Also(
			m.Validate(ctx).ViaFieldIndex("mappings", i).ViaField("spec"),
		)
		// TODO look for conflicting names
	}

	if b.Status.Annotations != nil {
		errs = errs.Also(
			apis.ErrDisallowedFields("status.annotations"),
		)
	}

	return errs
}

func (m Mapping) Validate(ctx context.Context) (errs *apis.FieldError) {
	if m.Name == "" {
		errs = errs.Also(
			apis.ErrMissingField("name"),
		)
	}

	return errs
}

func (b *ServiceBinding) SetDefaults(context.Context) {
	if b.Spec.Name == "" {
		b.Spec.Name = b.Name
	}
	if b.Spec.Application.Namespace == "" {
		// Default the application's namespace to our namespace.
		b.Spec.Application.Namespace = b.Namespace
	}
	if b.Spec.Service.Namespace == "" {
		// Default the service's namespace to our namespace.
		b.Spec.Service.Namespace = b.Namespace
	}
}

func (b *ServiceBinding) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ServiceBinding")
}
