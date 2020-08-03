/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	"fmt"

	servicebindingv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha2"
)

func ProjectedSecret(binding *servicebindingv1alpha2.ServiceBinding) string {
	name := binding.Name
	// limit the returned value to at most 63 characters
	if len(name) > 52 {
		name = name[:52]
	}
	return fmt.Sprintf("%s-projection", name)
}
