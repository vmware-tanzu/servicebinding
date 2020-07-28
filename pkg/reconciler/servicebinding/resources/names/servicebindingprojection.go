/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	servicebindingv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha2"
)

func ServiceBindingProjection(binding *servicebindingv1alpha2.ServiceBinding) string {
	return binding.Name
}
