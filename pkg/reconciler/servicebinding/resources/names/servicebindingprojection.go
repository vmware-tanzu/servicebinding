/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	servicebindingv1alpha3 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha3"
)

func ServiceBindingProjection(binding *servicebindingv1alpha3.ServiceBinding) string {
	return binding.Name
}
