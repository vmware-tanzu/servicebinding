/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package servicebindingprojection

import (
	"context"
	"fmt"
	"testing"

	servicebindinginternalv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebindinginternal/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	clientgotesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/client/injection/ducks/duck/v1/podspecable"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	dynamicclient "knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/tracker"
	"knative.dev/pkg/webhook/psbinding"

	// register injection fakes
	_ "github.com/vmware-labs/service-bindings/pkg/client/injection/informers/servicebindinginternal/v1alpha2/servicebindingprojection/fake"
	_ "knative.dev/pkg/client/injection/ducks/duck/v1/podspecable/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/namespace/fake"
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
	name := "my-service"
	key := fmt.Sprintf("%s/%s", namespace, name)

	table := TableTest{{
		Name: "bad workqueue key",
		Key:  "too/many/parts",
	}, {
		Name: "key not found",
		Key:  key,
	}, {
		Name: "nop - in sync",
		Key:  key,
		Objects: []runtime.Object{
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						"bindings.knative.dev/include": "true",
					},
				},
			},
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Finalizers: []string{
						"servicebindingprojections.internal.service.binding",
					},
					Generation: 1,
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name: name,
					Application: servicebindinginternalv1alpha2.ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-application",
						},
					},
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
				Status: servicebindinginternalv1alpha2.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionApplicationAvailable,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "my-application",
					Annotations: map[string]string{
						"internal.service.binding/projection-e9ead9b18f311f72f9c7a54af76": "my-secret",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "my-secret",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, {
		Name: "nop - ignore custom projections",
		Key:  key,
		Objects: []runtime.Object{
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Annotations: map[string]string{
						"projection.service.binding/type": "Custom",
					},
					Generation: 1,
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name: name,
					Application: servicebindinginternalv1alpha2.ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-application",
						},
					},
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
		},
	}, {
		Name: "undo previous binding that is now custom",
		Key:  key,
		Objects: []runtime.Object{
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						"bindings.knative.dev/include": "true",
					},
				},
			},
			&servicebindinginternalv1alpha2.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Annotations: map[string]string{
						"projection.service.binding/type": "Custom",
					},
					Finalizers: []string{
						"servicebindingprojections.internal.service.binding",
					},
					Generation: 1,
				},
				Spec: servicebindinginternalv1alpha2.ServiceBindingProjectionSpec{
					Name: name,
					Application: servicebindinginternalv1alpha2.ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-application",
						},
					},
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
				Status: servicebindinginternalv1alpha2.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionApplicationAvailable,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "my-application",
					Annotations: map[string]string{
						"internal.service.binding/projection-e9ead9b18f311f72f9c7a54af76": "my-secret",
					},
				},
				// will also remove injected PodTemplateSpec items, but the
				// patch bytes are not ordered deterministically so we're
				// artificially limiting the patch to a single item.
			},
		},
		WantPatches: []clientgotesting.PatchActionImpl{
			{
				ActionImpl: clientgotesting.ActionImpl{
					Namespace: namespace,
				},
				Name:      "my-application",
				PatchType: types.JSONPatchType,
				Patch:     []byte(`[{"op":"remove","path":"/metadata/annotations"}]`),
			},
			{
				ActionImpl: clientgotesting.ActionImpl{
					Namespace: namespace,
				},
				Name:      name,
				PatchType: types.MergePatchType,
				Patch:     []byte(`{"metadata":{"finalizers":[],"resourceVersion":""}}`),
			},
		},
	}}

	table.Test(t, MakeFactory(func(ctx context.Context, listers *Listers, cmw configmap.Watcher) controller.Reconciler {
		ctx = podspecable.WithDuck(ctx)

		c := &ConditionalReconciler{
			Delegate: &psbinding.BaseReconciler{
				GVR: servicebindinginternalv1alpha2.SchemeGroupVersion.WithResource("servicebindingprojections"),
				Get: func(namespace string, name string) (psbinding.Bindable, error) {
					return listers.GetServiceBindingProjectionLister().ServiceBindingProjections(namespace).Get(name)
				},
				DynamicClient: dynamicclient.Get(ctx),
				Recorder: record.NewBroadcaster().NewRecorder(
					scheme.Scheme, corev1.EventSource{Component: controllerAgentName}),
				NamespaceLister: listers.GetNamespaceLister(),
				Tracker:         GetTracker(ctx),
				Factory:         podspecable.Get(ctx),
			},
			Lister: listers.GetServiceBindingProjectionLister(),
		}
		return c
	}))
}
