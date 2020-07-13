/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	"fmt"

	servicev1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/service/v1alpha2"
)

func ProjectedSecret(binding *servicev1alpha2.ServiceBinding) string {
	// TODO generate the secret name
	return fmt.Sprintf("%s-projection", binding.Name)
}
