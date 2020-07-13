/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
)

// +genduck

type Imageable struct {
	LatestImage string `json:"latestImage,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImagableType is a skeleton type wrapping Imageable in the manner we expect
// resource writers defining compatible resources to embed it.  We will
// typically use this type to deserialize Imageable ObjectReferences and
// access the Imageable data.  This is not a real resource.
type ImageableType struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status Imageable `json:"status"`
}

func (*ImageableType) GetListType() runtime.Object {
	return &ImageableTypeList{}
}

func (t *ImageableType) Populate() {
	t.Status = Imageable{
		LatestImage: "some/image",
	}
}

var (
	// Verify AddressableType resources meet duck contracts.
	_ duck.Populatable = (*ImageableType)(nil)
	_ apis.Listable    = (*ImageableType)(nil)
)

func (*Imageable) GetFullType() duck.Populatable {
	return &ImageableType{}
}

var (
	// Addressable is an Implementable "duck type".
	//_ duck.Implementable = (*Imageable)(nil)
	_ duck.Implementable = (*Imageable)(nil)
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddressableTypeList is a list of AddressableType resources
type ImageableTypeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ImageableType `json:"items"`
}
