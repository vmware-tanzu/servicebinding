/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"testing"

	rduckv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/duck/v1alpha1"
	"knative.dev/pkg/apis/duck"
)

func TestImplementsBinding(t *testing.T) {
	if err := duck.VerifyType(&ProvisionedService{}, &rduckv1alpha1.Serviceable{}); err != nil {
		t.Fatal(err)
	}
}
