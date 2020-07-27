/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	servicev1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/service/v1alpha2"
)

func ServiceBindingProjection(binding *servicev1alpha2.ServiceBinding) string {
	return binding.Name
}
