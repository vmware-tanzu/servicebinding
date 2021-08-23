/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"

	labsinternalv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labsinternal/v1alpha1"
	servicebindingv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha2"
	resourcenames "github.com/vmware-labs/service-bindings/pkg/reconciler/servicebinding/resources/names"
)

func MakeServiceBindingProjection(binding *servicebindingv1alpha2.ServiceBinding) (*labsinternalv1alpha1.ServiceBindingProjection, error) {
	projection := &labsinternalv1alpha1.ServiceBindingProjection{
		ObjectMeta: metav1.ObjectMeta{
			Name:        resourcenames.ServiceBindingProjection(binding),
			Namespace:   binding.Namespace,
			Annotations: map[string]string{},
			Labels: kmeta.UnionMaps(binding.GetLabels(), map[string]string{
				servicebindingv1alpha2.ServiceBindingLabelKey: binding.Name,
			}),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(binding)},
		},
		Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
			Name:     binding.Spec.Name,
			Type:     binding.Spec.Type,
			Provider: binding.Spec.Provider,
			Binding:  *binding.Status.Binding,
			Workload: *binding.Spec.Workload,
			Env:      binding.Spec.Env,
		},
	}

	for k, v := range binding.Annotations {
		// copy forward "serice.bindings" annotations
		if strings.Contains(k, servicebindingv1alpha2.GroupName) {
			projection.Annotations[k] = v
		}
	}

	return projection, nil
}
