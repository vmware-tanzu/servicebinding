/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	servicebindingv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
)

func TestMakeProjectedSecret(t *testing.T) {
	tests := []struct {
		name        string
		binding     *servicebindingv1alpha2.ServiceBinding
		reference   *corev1.Secret
		expected    *corev1.Secret
		expectedErr bool
	}{
		{
			name: "empty",
			binding: &servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
			},
			reference: &corev1.Secret{},
			expected: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               "my-binding",
							Controller:         ptr.Bool(true),
							BlockOwnerDeletion: ptr.Bool(true),
						},
					},
				},
			},
		},
		{
			name: "preserve existing keys in referenced secret",
			binding: &servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
			},
			reference: &corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("root"),
					"password": []byte("password1"),
				},
			},
			expected: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               "my-binding",
							Controller:         ptr.Bool(true),
							BlockOwnerDeletion: ptr.Bool(true),
						},
					},
				},
				Data: map[string][]byte{
					"username": []byte("root"),
					"password": []byte("password1"),
				},
			},
		},
		{
			name: "add type and provider to the projected secret",
			binding: &servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Type:     "mysql",
					Provider: "bitnami",
				},
			},
			reference: &corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("root"),
					"password": []byte("password1"),
				},
			},
			expected: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               "my-binding",
							Controller:         ptr.Bool(true),
							BlockOwnerDeletion: ptr.Bool(true),
						},
					},
				},
				Data: map[string][]byte{
					"username": []byte("root"),
					"password": []byte("password1"),
					"type":     []byte("mysql"),
					"provider": []byte("bitnami"),
				},
			},
		},
		{
			name: "add mappings to the projected secret",
			binding: &servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Mappings: []servicebindingv1alpha2.Mapping{
						{
							Name:  "literal",
							Value: "value",
						},
						{
							Name:  "templated",
							Value: "{{ .username }}:{{ .password }}",
						},
					},
				},
			},
			reference: &corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("root"),
					"password": []byte("password1"),
				},
			},
			expected: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               "my-binding",
							Controller:         ptr.Bool(true),
							BlockOwnerDeletion: ptr.Bool(true),
						},
					},
				},
				Data: map[string][]byte{
					"username":  []byte("root"),
					"password":  []byte("password1"),
					"literal":   []byte("value"),
					"templated": []byte("root:password1"),
				},
			},
		},
		{
			name: "invalid mapping template",
			binding: &servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Mappings: []servicebindingv1alpha2.Mapping{
						{
							Name:  "bad-template",
							Value: "{{  }",
						},
					},
				},
			},
			reference: &corev1.Secret{
				Data: map[string][]byte{
					"username": []byte("root"),
					"password": []byte("password1"),
				},
			},
			expectedErr: true,
		},
		{
			name: "error applying mapping template",
			binding: &servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Mappings: []servicebindingv1alpha2.Mapping{
						{
							Name:  "bad-template",
							Value: "{{ call .invalid }}",
						},
					},
				},
			},
			reference: &corev1.Secret{
				Data: map[string][]byte{},
			},
			expectedErr: true,
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			binding := c.binding.DeepCopy()
			reference := c.reference.DeepCopy()
			actual, err := MakeProjectedSecret(c.binding, c.reference)
			if actualErr := err == nil; actualErr == c.expectedErr {
				if c.expectedErr {
					t.Errorf("%s: MakeProjectedSecret() expected error", c.name)
				} else {
					t.Errorf("%s: MakeProjectedSecret() expected no error, got %v", c.name, err)
				}
			}
			if diff := cmp.Diff(c.expected, actual); diff != "" {
				t.Errorf("%s: MakeProjectedSecret() (-expected, +actual): %s", c.name, diff)
			}
			if diff := cmp.Diff(c.binding, binding); diff != "" {
				t.Errorf("%s: MakeProjectedSecret() unexpected binding mutation (-expected, +actual): %s", c.name, diff)
			}
			if diff := cmp.Diff(c.reference, reference); diff != "" {
				t.Errorf("%s: MakeProjectedSecret() unexpected reference mutation (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}
