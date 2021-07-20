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
	labsinternalv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labsinternal/v1alpha1"
	servicebindingv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha2"
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
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/ptr"
	"knative.dev/pkg/tracker"

	// register injection fakes
	_ "github.com/vmware-labs/service-bindings/pkg/client/injection/ducks/duck/v1alpha2/serviceable/fake"
	_ "github.com/vmware-labs/service-bindings/pkg/client/injection/informers/labsinternal/v1alpha1/servicebindingprojection/fake"
	_ "github.com/vmware-labs/service-bindings/pkg/client/injection/informers/servicebinding/v1alpha2/servicebinding/fake"
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
	secretName := "my-secret"
	provisionedService := &labsv1alpha1.ProvisionedService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "my-service",
		},
		Status: labsv1alpha1.ProvisionedServiceStatus{
			Binding: corev1.LocalObjectReference{
				Name: secretName,
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
						Name: secretName,
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
			&labsinternalv1alpha1.ServiceBindingProjection{
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
				Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: secretName,
					},
				},
				Status: labsinternalv1alpha1.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   labsinternalv1alpha1.ServiceBindingProjectionConditionReady,
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
		Name: "creates servicebindingprojection",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
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
						Name: secretName,
					},
				},
			},
		},
		WantCreates: []runtime.Object{
			&labsinternalv1alpha1.ServiceBindingProjection{
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
				Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: secretName,
					},
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
						Name: secretName,
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
			Eventf(corev1.EventTypeNormal, "Created", "Created ServiceBindingProjection %q", name),
			Eventf(corev1.EventTypeNormal, "Reconciled", "ServiceBinding reconciled: %q", key),
		},
	}, {
		Name: "updates servicebindingprojection",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
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
						Name: secretName,
					},
				},
			},
			&labsinternalv1alpha1.ServiceBindingProjection{
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
				Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: secretName,
					},
				},
			},
		},
		WantUpdates: []clientgotesting.UpdateActionImpl{
			{
				Object: &labsinternalv1alpha1.ServiceBindingProjection{
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
					Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
						Name:        name,
						Application: applicationRef,
						Binding: corev1.LocalObjectReference{
							Name: secretName,
						},
					},
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
						Name: secretName,
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
						Name: secretName,
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
			&labsinternalv1alpha1.ServiceBindingProjection{
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
				Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: secretName,
					},
				},
				Status: labsinternalv1alpha1.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   labsinternalv1alpha1.ServiceBindingProjectionConditionReady,
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
		Name: "error creating servicebindingprojection",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
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
						Name: secretName,
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
		},
		WithReactors: []clientgotesting.ReactionFunc{
			InduceFailure("create", "servicebindingprojections"),
		},
		WantErr: true,
		WantCreates: []runtime.Object{
			&labsinternalv1alpha1.ServiceBindingProjection{
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
				Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: secretName,
					},
				},
			},
		},
		WantEvents: []string{
			Eventf(corev1.EventTypeWarning, "CreationFailed", "Failed to create ServiceBindingProjection %q: inducing failure for create servicebindingprojections", name),
			Eventf(corev1.EventTypeWarning, "InternalError", "failed to create ServiceBindingProjection: inducing failure for create servicebindingprojections"),
		},
	}, {
		Name: "error updating servicebindingprojection",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
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
						Name: secretName,
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
			&labsinternalv1alpha1.ServiceBindingProjection{
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
				Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: secretName,
					},
				},
				Status: labsinternalv1alpha1.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   labsinternalv1alpha1.ServiceBindingProjectionConditionReady,
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
				Object: &labsinternalv1alpha1.ServiceBindingProjection{
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
					Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
						Name:        name,
						Application: applicationRef,
						Binding: corev1.LocalObjectReference{
							Name: secretName,
						},
					},
					Status: labsinternalv1alpha1.ServiceBindingProjectionStatus{
						Status: duckv1.Status{
							Conditions: duckv1.Conditions{
								{
									Type:   labsinternalv1alpha1.ServiceBindingProjectionConditionReady,
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
		Name: "error servicebindingprojection is not owned by us",
		Key:  key,
		Objects: []runtime.Object{
			provisionedService.DeepCopy(),
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
						Name: secretName,
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
			&labsinternalv1alpha1.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
				},
				Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
					Name:        name,
					Application: applicationRef,
					Binding: corev1.LocalObjectReference{
						Name: secretName,
					},
				},
				Status: labsinternalv1alpha1.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   labsinternalv1alpha1.ServiceBindingProjectionConditionReady,
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
			bindingclient:                  servicebindingsclient.Get(ctx),
			resolver:                       resolver.NewServiceableResolver(ctx, func(types.NamespacedName) {}),
			serviceBindingProjectionLister: listers.GetServiceBindingProjectionLister(),
			tracker:                        GetTracker(ctx),
		}

		return servicebindingreconciler.NewReconciler(ctx, logging.FromContext(ctx), servicebindingsclient.Get(ctx),
			listers.GetServiceBindingLister(), controller.GetEventRecorder(ctx), r)
	}))
}
