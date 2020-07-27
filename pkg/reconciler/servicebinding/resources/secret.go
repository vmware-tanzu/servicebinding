/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	"bytes"
	"fmt"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/kmeta"

	servicev1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/service/v1alpha2"
	resourcenames "github.com/vmware-labs/service-bindings/pkg/reconciler/servicebinding/resources/names"
)

func MakeProjectedSecret(binding *servicev1alpha2.ServiceBinding, reference *corev1.Secret) (*corev1.Secret, error) {
	projection := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourcenames.ProjectedSecret(binding),
			Namespace: binding.Namespace,
			Labels: kmeta.UnionMaps(binding.GetLabels(), map[string]string{
				servicev1alpha2.ServiceBindingLabelKey: binding.Name,
			}),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(binding)},
		},
	}

	projection.Data = reference.DeepCopy().Data
	if binding.Spec.Type != "" {
		projection.Data["type"] = []byte(binding.Spec.Type)
	}
	if binding.Spec.Provider != "" {
		projection.Data["provider"] = []byte(binding.Spec.Provider)
	}
	for _, m := range binding.Spec.Mappings {
		t, err := template.New("").Parse(m.Value)
		if err != nil {
			return nil, fmt.Errorf("Invalid template for mapping %s: %w", m.Name, err)
		}
		buf := bytes.NewBuffer([]byte{})
		// convert map[string][]byte to map[string]string
		values := make(map[string]string, len(projection.Data))
		for k, v := range projection.Data {
			values[k] = string(v)
		}
		err = t.Execute(buf, values)
		if err != nil {
			return nil, fmt.Errorf("Error executing template for mapping %s: %w", m.Name, err)
		}
		projection.Data[m.Name] = buf.Bytes()
	}

	return projection, nil
}
