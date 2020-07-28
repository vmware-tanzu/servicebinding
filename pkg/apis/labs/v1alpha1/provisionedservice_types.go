/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

const (
	ProvisionedServiceAnnotationKey = GroupName + "/provisioned-service"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ProvisionedService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProvisionedServiceSpec   `json:"spec,omitempty"`
	Status ProvisionedServiceStatus `json:"status,omitempty"`
}

var (
	// Check that ProvisionedService can be validated and defaulted.
	_ apis.Validatable   = (*ProvisionedService)(nil)
	_ apis.Defaultable   = (*ProvisionedService)(nil)
	_ kmeta.OwnerRefable = (*ProvisionedService)(nil)
	_ duckv1.KRShaped    = (*ProvisionedService)(nil)
)

type ProvisionedServiceSpec struct {
	Binding corev1.LocalObjectReference `json:"binding,omitempty"`
}

type ProvisionedServiceStatus struct {
	duckv1.Status `json:",inline"`
	Binding       corev1.LocalObjectReference `json:"binding,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProvisionedServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProvisionedService `json:"items"`
}

func (p *ProvisionedService) Validate(ctx context.Context) (errs *apis.FieldError) {
	if p.Spec.Binding.Name == "" {
		errs = errs.Also(
			apis.ErrMissingField("spec.binding.name"),
		)
	}

	return errs
}

func (p *ProvisionedService) SetDefaults(context.Context) {
	// nothing to do
}

func (p *ProvisionedService) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ProvisionedService")
}
