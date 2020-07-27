/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"

	servicev1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/service/v1alpha2"
	serviceinternalv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/serviceinternal/v1alpha2"
	resourcenames "github.com/vmware-labs/service-bindings/pkg/reconciler/servicebinding/resources/names"
)

func MakeServiceBindingProjection(binding *servicev1alpha2.ServiceBinding) (*serviceinternalv1alpha2.ServiceBindingProjection, error) {
	projection := &serviceinternalv1alpha2.ServiceBindingProjection{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourcenames.ServiceBindingProjection(binding),
			Namespace: binding.Namespace,
			Labels: kmeta.UnionMaps(binding.GetLabels(), map[string]string{
				servicev1alpha2.ServiceBindingLabelKey: binding.Name,
			}),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(binding)},
		},
		Spec: serviceinternalv1alpha2.ServiceBindingProjectionSpec{
			Name:        binding.Spec.Name,
			Binding:     *binding.Status.Binding,
			Application: *binding.Spec.Application,
			Env:         binding.Spec.Env,
		},
	}

	return projection, nil
}
