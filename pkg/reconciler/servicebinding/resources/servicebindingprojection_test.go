/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	servicebindingv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha2"
	servicebindinginternalv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebindinginternal/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/tracker"
)

func TestMakeServiceBindingProjection(t *testing.T) {
	True := true
	tests := []struct {
		name        string
		binding     *servicebindingv1alpha2.ServiceBinding
		expected    *servicebindinginternalv1alpha2.ServiceBindingProjection
		expectedErr bool
	}{
		{
			name: "project binding",
			binding: &servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name: "my-binding",
					Application: &servicebindingv1alpha2.ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
					},
					Env: []servicebindingv1alpha2.EnvVar{
						{
							Name: "MY_VAR",
							Key:  "my-key",
						},
					},
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
			expected: &servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               "my-binding",
							Controller:         &True,
							BlockOwnerDeletion: &True,
						},
					},
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name: "my-binding",
					Application: servicebindinginternalv1alpha2.ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
					},
					Env: []servicebindinginternalv1alpha2.EnvVar{
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
