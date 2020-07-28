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
	"k8s.io/apimachinery/pkg/util/intstr"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/tracker"
	"knative.dev/pkg/webhook/psbinding"
)

const (
	ServiceBindingProjectionAnnotationKey = GroupName + "/service-binding-projection"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ServiceBindingProjection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceBindingProjectionSpec   `json:"spec,omitempty"`
	Status ServiceBindingProjectionStatus `json:"status,omitempty"`
}

var (
	// Check that ServiceBinding can be validated and defaulted.
	_ apis.Validatable   = (*ServiceBindingProjection)(nil)
	_ apis.Defaultable   = (*ServiceBindingProjection)(nil)
	_ kmeta.OwnerRefable = (*ServiceBindingProjection)(nil)
	_ duckv1.KRShaped    = (*ServiceBindingProjection)(nil)

	// Check is Bindable
	_ psbinding.Bindable  = (*ServiceBindingProjection)(nil)
	_ duck.BindableStatus = (*ServiceBindingProjectionStatus)(nil)
)

type ServiceBindingProjectionSpec struct {
	// Name of the service binding on disk, defaults to this resource's name
	Name string `json:"name"`

	// Binding reference to the service binding's projected secret
	Binding corev1.LocalObjectReference `json:"binding"`

	// Application resource to inject the binding into
	Application ApplicationReference `json:"application"`

	// Env projects keys from the binding secret into the application as
	// environment variables
	Env []EnvVar `json:"env,omitempty"`
}

type ApplicationReference struct {
	tracker.Reference

	// Containers to target within the application. If not set, all containers
	// will be injected. Containers may be specified by index or name.
	// InitContainers may only be specified by name.
	Containers []intstr.IntOrString `json:"containers,omitempty"`
}

type EnvVar struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type ServiceBindingProjectionStatus struct {
	duckv1.Status `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ServiceBindingProjectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceBindingProjection `json:"items"`
}

func (b *ServiceBindingProjection) Validate(ctx context.Context) (errs *apis.FieldError) {
	if b.Spec.Name == "" {
		errs = errs.Also(
			apis.ErrMissingField("spec.name"),
		)
	}

	if b.Spec.Binding.Name == "" {
		errs = errs.Also(
			apis.ErrMissingField("spec.binding"),
		)
	}

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

	if b.Status.Annotations != nil {
		errs = errs.Also(
			apis.ErrDisallowedFields("status.annotations"),
		)
	}

	return errs
}

func (e EnvVar) Validate(ctx context.Context) (errs *apis.FieldError) {
	if e.Name == "" {
		errs = errs.Also(
			apis.ErrMissingField("name"),
		)
	}
	if e.Key == "" {
		errs = errs.Also(
			apis.ErrMissingField("key"),
		)
	}

	return errs
}

func (b *ServiceBindingProjection) SetDefaults(context.Context) {
	// no defaults to apply
}

func (b *ServiceBindingProjection) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ServiceBindingProjection")
}
