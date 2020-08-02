/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package servicebinding

import (
	"context"
	"fmt"
	"testing"
	"time"

	labsv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labs/v1alpha1"
	servicebindingv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha2"
	servicebindinginternalv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebindinginternal/v1alpha2"
	servicebindingsclient "github.com/vmware-labs/service-bindings/pkg/client/injection/client"
	"github.com/vmware-labs/service-bindings/pkg/client/injection/ducks/duck/v1alpha2/serviceable"
	servicebindingreconciler "github.com/vmware-labs/service-bindings/pkg/client/injection/reconciler/servicebinding/v1alpha2/servicebinding"
	"github.com/vmware-labs/service-bindings/pkg/resolver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgotesting "k8s.io/client-go/testing"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/tracker"

	// register injection fakes
	_ "github.com/vmware-labs/service-bindings/pkg/client/injection/ducks/duck/v1alpha2/serviceable/fake"
	_ "github.com/vmware-labs/service-bindings/pkg/client/injection/informers/servicebinding/v1alpha2/servicebinding/fake"
	_ "github.com/vmware-labs/service-bindings/pkg/client/injection/informers/servicebindinginternal/v1alpha2/servicebindingprojection/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/secret/fake"
	_ "knative.dev/pkg/injection/clients/dynamicclient/fake"

	. "github.com/vmware-labs/service-bindings/pkg/reconciler/testing"
	. "knative.dev/pkg/reconciler/testing"
)

func TestNewController(t *testing.T) {
	ctx, _ := SetupFakeContext(t)

	c := NewController(ctx, configmap.NewStaticWatcher())

	if c == nil {
		t.Fatal("expected NewController to return a non-nil value")
	}
}

func TestReconcile(t *testing.T) {
	namespace := "my-namespace"
	name := "my-binding"
	key := fmt.Sprintf("%s/%s", namespace, name)
	now := metav1.NewTime(time.Now())
	serviceSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "my-secret",
		},
		Data: map[string][]byte{
			"username": []byte("root"),
			"password": []byte("password!"),
		},
	}
	provisionedService := &labsv1alpha1.ProvisionedService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "my-service",
		},
		Status: labsv1alpha1.ProvisionedServiceStatus{
			Binding: corev1.LocalObjectReference{
				Name: serviceSecret.Name,
			},
		},
	}
	serviceRef := tracker.Reference{
		APIVersion: provisionedService.GetGroupVersionKind().GroupVersion().String(),
		Kind:       provisionedService.GetGroupVersionKind().Kind,
		Name:       provisionedService.Name,
	}
	applicationRef := servicebindingv1alpha2.ApplicationReference{
		Reference: tracker.Reference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "my-application",
		},
	}

	table := TableTest{{
		Name: "bad workqueue key",
		Key:  "too/many/parts",
	}, {
		Name: "key not found",
		Key:  key,
	}, {
		Name: "nop - deleted",
		Key:  key,
		Objects: []runtime.Object{
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         namespace,
					Name:              name,
					DeletionTimestamp: &now,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{},
			},
		},
	}, {
		Name: "nop - in sync",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
			serviceSecret.DeepCopy(),
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionProjectionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionServiceAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name + "-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Data: serviceSecret.Data,
			},
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
				Status: servicebindinginternalv1alpha2.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		WantEvents: []string{
			Eventf(corev1.EventTypeNormal, "Reconciled", "ServiceBinding reconciled: %q", key),
		},
	}, {
		Name: "creates projected secret and servicebindingprojection",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
			serviceSecret.DeepCopy(),
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
			},
		},
		WantCreates: []runtime.Object{
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name + "-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Data: serviceSecret.Data,
			},
		},
		WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
			Object: &servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionProjectionReady,
								Status: corev1.ConditionUnknown,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionReady,
								Status: corev1.ConditionUnknown,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionServiceAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		}},
		WantEvents: []string{
			Eventf(corev1.EventTypeNormal, "Created", "Created projected Secret %q", name+"-projection"),
			Eventf(corev1.EventTypeNormal, "Created", "Created ServiceBindingProjection %q", name),
			Eventf(corev1.EventTypeNormal, "Reconciled", "ServiceBinding reconciled: %q", key),
		},
	}, {
		Name: "updates projected secret and servicebindingprojection",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
			serviceSecret.DeepCopy(),
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
			},
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name + "-projection",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Data: serviceSecret.Data,
			},
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{
			{
				Object: &servicebindinginternalv1alpha2.ServiceBindingProjection{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      name,
						Labels: map[string]string{
							"service.binding/servicebinding": "my-binding",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "service.binding/v1alpha2",
								Kind:               "ServiceBinding",
								Name:               name,
								BlockOwnerDeletion: ptr.Bool(true),
								Controller:         ptr.Bool(true),
							},
						},
					},
					Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
						Name:        name,
						Application: applicationRef,
						Binding: corev1.LocalObjectReference{
							Name: name + "-projection",
						},
					},
				},
			},
			{
				Object: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      name + "-projection",
						Labels: map[string]string{
							"service.binding/servicebinding": "my-binding",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "service.binding/v1alpha2",
								Kind:               "ServiceBinding",
								Name:               name,
								BlockOwnerDeletion: ptr.Bool(true),
								Controller:         ptr.Bool(true),
							},
						},
					},
					Data: serviceSecret.Data,
				},
			},
		},
		WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
			Object: &servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionProjectionReady,
								Status: corev1.ConditionUnknown,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionReady,
								Status: corev1.ConditionUnknown,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionServiceAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		}},
		WantEvents: []string{
			Eventf(corev1.EventTypeNormal, "Reconciled", "ServiceBinding reconciled: %q", key),
		},
	}, {
		Name: "missing referenced service",
		Key:  key,
		Objects: []runtime.Object{
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionProjectionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionServiceAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name + "-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Data: serviceSecret.Data,
			},
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
				Status: servicebindinginternalv1alpha2.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		WantErr: true,
		WantEvents: []string{
			Eventf(corev1.EventTypeWarning, "InternalError", "failed to get resource for bindings.labs.vmware.com/v1alpha1, Resource=provisionedservices: provisionedservices.bindings.labs.vmware.com %q not found", serviceRef.Name),
		},
	}, {
		Name: "error creating projected secret",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
			serviceSecret.DeepCopy(),
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionProjectionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionServiceAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
				Status: servicebindinginternalv1alpha2.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		WithReactors: []clientgotesting.ReactionFunc{
			InduceFailure("create", "secrets"),
		},
		WantErr: true,
		WantCreates: []runtime.Object{
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name + "-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Data: serviceSecret.Data,
			},
		},
		WantEvents: []string{
			Eventf(corev1.EventTypeWarning, "CreationFailed", "Failed to create projected Secret %q: inducing failure for create secrets", name+"-projection"),
			Eventf(corev1.EventTypeWarning, "InternalError", "failed to create projected Secret: inducing failure for create secrets"),
		},
	}, {
		Name: "error creating servicebindingprojection",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
			serviceSecret.DeepCopy(),
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionProjectionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionServiceAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name + "-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Data: serviceSecret.Data,
			},
		},
		WithReactors: []clientgotesting.ReactionFunc{
			InduceFailure("create", "servicebindingprojections"),
		},
		WantErr: true,
		WantCreates: []runtime.Object{
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
			},
		},
		WantEvents: []string{
			Eventf(corev1.EventTypeWarning, "CreationFailed", "Failed to create ServiceBindingProjection %q: inducing failure for create servicebindingprojections", name),
			Eventf(corev1.EventTypeWarning, "InternalError", "failed to create ServiceBindingProjection: inducing failure for create servicebindingprojections"),
		},
	}, {
		Name: "error updating projected secret",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
			serviceSecret.DeepCopy(),
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionProjectionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionServiceAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name + "-projection",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Data: serviceSecret.Data,
			},
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
				Status: servicebindinginternalv1alpha2.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		WithReactors: []clientgotesting.ReactionFunc{
			InduceFailure("update", "secrets"),
		},
		WantErr: true,
		WantUpdates: []clientgotesting.UpdateActionImpl{
			{
				Object: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      name + "-projection",
						Labels: map[string]string{
							"service.binding/servicebinding": "my-binding",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "service.binding/v1alpha2",
								Kind:               "ServiceBinding",
								Name:               name,
								BlockOwnerDeletion: ptr.Bool(true),
								Controller:         ptr.Bool(true),
							},
						},
					},
					Data: serviceSecret.Data,
				},
			},
		},
		WantEvents: []string{
			Eventf(corev1.EventTypeWarning, "InternalError", "failed to reconcile projected Secret: inducing failure for update secrets"),
		},
	}, {
		Name: "error updating servicebidningprojection",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
			serviceSecret.DeepCopy(),
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionProjectionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionServiceAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name + "-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Data: serviceSecret.Data,
			},
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
				Status: servicebindinginternalv1alpha2.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		WithReactors: []clientgotesting.ReactionFunc{
			InduceFailure("update", "servicebindingprojections"),
		},
		WantErr: true,
		WantUpdates: []clientgotesting.UpdateActionImpl{
			{
				Object: &servicebindinginternalv1alpha2.ServiceBindingProjection{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespace,
						Name:      name,
						Labels: map[string]string{
							"service.binding/servicebinding": "my-binding",
						},
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "service.binding/v1alpha2",
								Kind:               "ServiceBinding",
								Name:               name,
								BlockOwnerDeletion: ptr.Bool(true),
								Controller:         ptr.Bool(true),
							},
						},
					},
					Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
						Name:        name,
						Application: applicationRef,
						Binding: corev1.LocalObjectReference{
							Name: name + "-projection",
						},
					},
					Status: servicebindinginternalv1alpha2.ServiceBindingProjectionStatus{
						Status: duckv1.Status{
							Conditions: duckv1.Conditions{
								{
									Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
		},
		WantEvents: []string{
			Eventf(corev1.EventTypeWarning, "InternalError", "failed to reconcile ServiceBindingProjection: inducing failure for update servicebindingprojections"),
		},
	}, {
		Name: "error projected secret is not owned by us",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
			serviceSecret.DeepCopy(),
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionProjectionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionServiceAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name + "-projection",
				},
				Data: serviceSecret.Data,
			},
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
				Status: servicebindinginternalv1alpha2.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		WantErr: true,
		WantEvents: []string{
			Eventf(corev1.EventTypeWarning, "InternalError", "ServiceBinding %q does not own projected Secret: %q", name, name+"-projection"),
		},
	}, {
		Name: "error servicebindingprojection is not owned by us",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
			serviceSecret.DeepCopy(),
			&servicebindingv1alpha2.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: servicebindingv1alpha2.ServiceBindingSpec{
					Name:        name,
					Application: &applicationRef,
					Service:     &serviceRef,
				},
				Status: servicebindingv1alpha2.ServiceBindingStatus{
					Binding: &corev1.LocalObjectReference{
						Name: name + "-projection",
					},
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionProjectionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindingv1alpha2.ServiceBindingConditionServiceAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name + "-projection",
					Labels: map[string]string{
						"service.binding/servicebinding": "my-binding",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         "service.binding/v1alpha2",
							Kind:               "ServiceBinding",
							Name:               name,
							BlockOwnerDeletion: ptr.Bool(true),
							Controller:         ptr.Bool(true),
						},
					},
				},
				Data: serviceSecret.Data,
			},
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: name + "-projection",
					},
				},
				Status: servicebindinginternalv1alpha2.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		WantErr: true,
		WantEvents: []string{
			Eventf(corev1.EventTypeWarning, "InternalError", "ServiceBinding %q does not own ServiceBindingProjection: %q", name, name),
		},
	}}

	table.Test(t, MakeFactory(func(ctx context.Context, listers *Listers, cmw configmap.Watcher) controller.Reconciler {
		ctx = serviceable.WithDuck(ctx)

		r := &Reconciler{
			kubeclient:                     kubeclient.Get(ctx),
			bindingclient:                  servicebindingsclient.Get(ctx),
			resolver:                       resolver.NewServiceableResolver(ctx, func(types.NamespacedName) {}),
			secretLister:                   listers.GetSecretLister(),
			serviceBindingProjectionLister: listers.GetServiceBindingProjectionLister(),
			tracker:                        GetTracker(ctx),
		}

		return servicebindingreconciler.NewReconciler(ctx, logging.FromContext(ctx), servicebindingsclient.Get(ctx),
			listers.GetServiceBindingLister(), controller.GetEventRecorder(ctx), r)
	}))
}
