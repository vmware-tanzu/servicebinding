/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestRegisterHelpers(t *testing.T) {
	if got, want := Kind("Foo"), "Foo.bindings.labs.vmware.com"; got.String() != want {
		t.Errorf("Kind(Foo) = %v, want %v", got.String(), want)
	}

	if got, want := Resource("Foo"), "Foo.bindings.labs.vmware.com"; got.String() != want {
		t.Errorf("Resource(Foo) = %v, want %v", got.String(), want)
	}

	if got, want := SchemeGroupVersion.String(), "bindings.labs.vmware.com/v1alpha1"; got != want {
		t.Errorf("SchemeGroupVersion() = %v, want %v", got, want)
	}

	scheme := runtime.NewScheme()
	if err := addKnownTypes(scheme); err != nil {
		t.Errorf("addKnownTypes() = %v", err)
	}
}
