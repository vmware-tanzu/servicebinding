/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	labsinternalv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labsinternal/v1alpha1"
	servicebindingv1alpha3 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/tracker"
)

func TestMakeServiceBindingProjection(t *testing.T) {
	tests := []struct {
		name        string
		binding     *servicebindingv1alpha3.ServiceBinding
		expected    *labsinternalv1alpha1.ServiceBindingProjection
		expectedErr bool
	}{
		{
			name: "project binding",
			binding: &servicebindingv1alpha3.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
					Annotations: map[string]string{
						"servicebinding.io/include": "me",
						"ignore":                    "me",
					},
				},
				Spec: servicebindingv1alpha3.ServiceBindingSpec{
					Name:     "my-binding",
					Type:     "my-type",
					Provider: "my-provider",
					Workload: &servicebindingv1alpha3.WorkloadReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
					},
					Env: []servicebindingv1alpha3.EnvVar{
						{
							Name: "MY_VAR",
							Key:  "my-key",
						},
					},
				},
				Status: servicebindingv1alpha3.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
			expected: &labsinternalv1alpha1.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
					Annotations: map[string]string{
						"servicebinding.io/include": "me",
					},
					Labels: map[string]string{
						"servicebinding.io/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "servicebinding.io/v1alpha3",
							Kind:               "ServiceBinding",
							Name:               "my-binding",
							Controller:         ptr.Bool(true),
							BlockOwnerDeletion: ptr.Bool(true),
						},
					},
				},
				Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
					Name:     "my-binding",
					Type:     "my-type",
					Provider: "my-provider",
					Workload: labsinternalv1alpha1.WorkloadReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
					},
					Env: []labsinternalv1alpha1.EnvVar{
						{
							Name: "MY_VAR",
							Key:  "my-key",
						},
					},
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			binding := c.binding.DeepCopy()
			actual, err := MakeServiceBindingProjection(c.binding)
			if actualErr := err == nil; actualErr == c.expectedErr {
				if c.expectedErr {
					t.Errorf("%s: MakeServiceBindingProjection() expected error", c.name)
				} else {
					t.Errorf("%s: MakeServiceBindingProjection() expected no error, got %v", c.name, err)
				}
			}
			if diff := cmp.Diff(c.expected, actual); diff != "" {
				t.Errorf("%s: MakeServiceBindingProjection() (-expected, +actual): %s", c.name, diff)
			}
			if diff := cmp.Diff(c.binding, binding); diff != "" {
				t.Errorf("%s: MakeServiceBindingProjection() unexpected binding mutation (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}
