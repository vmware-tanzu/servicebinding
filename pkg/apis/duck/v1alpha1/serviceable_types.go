/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
)

// +genduck

type Serviceable struct {
	Binding corev1.LocalObjectReference `json:"binding"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImagableType is a skeleton type wrapping Serviceable in the manner we expect
// resource writers defining compatible resources to embed it.  We will
// typically use this type to deserialize Serviceable ObjectReferences and
// access the Serviceable data.  This is not a real resource.
type ServiceableType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status Serviceable `json:"status"`
}

func (*ServiceableType) GetListType() runtime.Object {
	return &ServiceableTypeList{}
}

func (t *ServiceableType) Populate() {
	t.Status = Serviceable{
		Binding: corev1.LocalObjectReference{Name: "my-secret"},
	}
}

var (
	// Verify AddressableType resources meet duck contracts.
	_ duck.Populatable = (*ServiceableType)(nil)
	_ apis.Listable    = (*ServiceableType)(nil)
)

func (*Serviceable) GetFullType() duck.Populatable {
	return &ServiceableType{}
}

var (
	// Addressable is an Implementable "duck type".
	//_ duck.Implementable = (*Serviceable)(nil)
	_ duck.Implementable = (*Serviceable)(nil)
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddressableTypeList is a list of AddressableType resources
type ServiceableTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ServiceableType `json:"items"`
}
