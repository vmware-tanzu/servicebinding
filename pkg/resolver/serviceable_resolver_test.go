/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package resolver

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	duckv1alpha3 "github.com/vmware-labs/service-bindings/pkg/apis/duck/v1alpha3"
	labsv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labs/v1alpha1"
	servicebindingv1alpha3 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha3"
	"github.com/vmware-labs/service-bindings/pkg/client/clientset/versioned/scheme"
	"github.com/vmware-labs/service-bindings/pkg/client/injection/ducks/duck/v1alpha3/serviceable"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakedynamicclient "knative.dev/pkg/injection/clients/dynamicclient/fake"
	"knative.dev/pkg/tracker"
)

func init() {
	// Add types to scheme
	appsv1.AddToScheme(scheme.Scheme)
	duckv1alpha3.AddToScheme(scheme.Scheme)
	labsv1alpha1.AddToScheme(scheme.Scheme)
	servicebindingv1alpha3.AddToScheme(scheme.Scheme)
}

func TestNewServiceableResolver(t *testing.T) {
	ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, []runtime.Object{}...)
	ctx = serviceable.WithDuck(ctx)
	r := NewServiceableResolver(ctx, func(types.NamespacedName) {})
	if r == nil {
		t.Fatal("expected NewServiceableResolver to return a non-nil value")
	}
}

func TestServiceableResolver_ServiceableFromObjectReference(t *testing.T) {
	tests := []struct {
		name        string
		seed        []runtime.Object
		ref         *tracker.Reference
		parent      interface{}
		expected    *corev1.LocalObjectReference
		expectedErr bool
	}{
		{
			name:        "empty",
			expectedErr: true,
		},
		{
			name: "lookup serviceable",
			seed: []runtime.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "bindings.labs.vmware.com/v1alpha1",
						"kind":       "ProvisionedService",
						"metadata": map[string]interface{}{
							"namespace": "my-namespace",
							"name":      "my-service",
						},
						"status": map[string]interface{}{
							"binding": map[string]interface{}{
								"name": "my-secret",
							},
						},
					},
				},
			},
			parent: &servicebindingv1alpha3.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
			},
			ref: &tracker.Reference{
				APIVersion: "bindings.labs.vmware.com/v1alpha1",
				Kind:       "ProvisionedService",
				Namespace:  "my-namespace",
				Name:       "my-service",
			},
			expected: &corev1.LocalObjectReference{
				Name: "my-secret",
			},
		},
		{
			name: "lookup secret",
			seed: []runtime.Object{},
			parent: &servicebindingv1alpha3.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
			},
			ref: &tracker.Reference{
				APIVersion: "v1",
				Kind:       "Secret",
				Namespace:  "my-namespace",
				Name:       "my-secret",
			},
			expected: &corev1.LocalObjectReference{
				Name: "my-secret",
			},
		},
		{
			name: "track error",
			seed: []runtime.Object{
				&unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "bindings.labs.vmware.com/v1alpha1",
						"kind":       "ProvisionedService",
						"metadata": map[string]interface{}{
							"namespace": "my-namespace",
							"name":      "my-service",
						},
						"status": map[string]interface{}{
							"binding": map[string]interface{}{
								"name": "my-secret",
							},
						},
					},
				},
			},
			parent: nil,
			ref: &tracker.Reference{
				APIVersion: "bindings.labs.vmware.com/v1alpha1",
				Kind:       "ProvisionedService",
				Namespace:  "my-namespace",
				Name:       "my-service",
			},
			expectedErr: true,
		},
		{
			name: "informer factory error",
			seed: []runtime.Object{
				&labsv1alpha1.ProvisionedService{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "my-namespace",
						Name:      "my-service",
					},
				},
			},
			parent: &servicebindingv1alpha3.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
			},
			ref: &tracker.Reference{
				APIVersion: "bindings.labs.vmware.com/v1alpha1",
				Kind:       "ProvisionedService",
				Namespace:  "my-namespace",
				Name:       "my-service",
			},
			expectedErr: true,
		},
		{
			name: "lookup error",
			seed: []runtime.Object{},
			parent: &servicebindingv1alpha3.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-namespace",
					Name:      "my-binding",
				},
			},
			ref: &tracker.Reference{
				APIVersion: "bindings.labs.vmware.com/v1alpha1",
				Kind:       "ProvisionedService",
				Namespace:  "my-namespace",
				Name:       "my-service",
			},
			expectedErr: true,
		},
		// TODO duckv1alpha1.ServiceableType is always returned, even if the fields are nil
		// {
		// 	name: "not serviceable",
		// 	seed: []runtime.Object{
		// 		&unstructured.Unstructured{
		// 			Object: map[string]interface{}{
		// 				"apiVersion": "apps/v1",
		// 				"kind":       "Deployment",
		// 				"metadata": map[string]interface{}{
		// 					"namespace": "my-namespace",
		// 					"name":      "my-service",
		// 				},
		// 			},
		// 		},
		// 	},
		// 	parent: &servicebindingv1alpha3.ServiceBinding{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Namespace: "my-namespace",
		// 			Name:      "my-binding",
		// 		},
		// 	},
		// 	ref: &tracker.Reference{
		// 		APIVersion: "apps/v1",
		// 		Kind:       "Deployment",
		// 		Namespace:  "my-namespace",
		// 		Name:       "my-service",
		// 	},
		// 	expectedErr: true,
		// },
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			ctx, _ := fakedynamicclient.With(context.Background(), scheme.Scheme, c.seed...)
			ctx = serviceable.WithDuck(ctx)
			r := NewServiceableResolver(ctx, func(types.NamespacedName) {})

			actual, err := r.ServiceableFromObjectReference(ctx, c.ref, c.parent)
			if actualErr := err == nil; actualErr == c.expectedErr {
				if c.expectedErr {
					t.Errorf("%s: ServiceableFromObjectReference() expected error", c.name)
				} else {
					t.Errorf("%s: ServiceableFromObjectReference() expected no error, got %v", c.name, err)
				}
			}
			if diff := cmp.Diff(c.expected, actual); diff != "" {
				t.Errorf("%s: ServiceableFromObjectReference() ref (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}
