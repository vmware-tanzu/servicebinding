/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package servicebindingprojection

import (
	"context"
	"fmt"
	"testing"

	labsinternalv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labsinternal/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/client/injection/ducks/duck/v1/podspecable"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	dynamicclient "knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/tracker"
	"knative.dev/pkg/webhook/psbinding"

	// register injection fakes
	_ "github.com/vmware-labs/service-bindings/pkg/client/injection/informers/labsinternal/v1alpha1/servicebindingprojection/fake"
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
			&labsinternalv1alpha1.ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Finalizers: []string{
						"servicebindingprojections.internal.bindings.labs.vmware.com",
					},
					Generation: 1,
				},
				Spec: labsinternalv1alpha1.ServiceBindingProjectionSpec{
					Name: name,
					Workload: labsinternalv1alpha1.WorkloadReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-workload",
						},
					},
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
				Status: labsinternalv1alpha1.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   labsinternalv1alpha1.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   labsinternalv1alpha1.ServiceBindingProjectionConditionWorkloadAvailable,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      "my-workload",
					Annotations: map[string]string{
						"internal.bindings.labs.vmware.com/projection-e9ead9b18f311f72f9c7a54af76427b50d02e2e3": "my-secret",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Projected: &corev1.ProjectedVolumeSource{
											Sources: []corev1.VolumeProjection{
												{
													Secret: &corev1.SecretProjection{
														LocalObjectReference: corev1.LocalObjectReference{
															Name: "my-secret",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}}

	table.Test(t, MakeFactory(func(ctx context.Context, listers *Listers, cmw configmap.Watcher) controller.Reconciler {
		ctx = podspecable.WithDuck(ctx)

		c := &psbinding.BaseReconciler{
			GVR: labsinternalv1alpha1.SchemeGroupVersion.WithResource("servicebindingprojections"),
			Get: func(namespace string, name string) (psbinding.Bindable, error) {
				return listers.GetServiceBindingProjectionLister().ServiceBindingProjections(namespace).Get(name)
			},
			DynamicClient: dynamicclient.Get(ctx),
			Recorder: record.NewBroadcaster().NewRecorder(
				scheme.Scheme, corev1.EventSource{Component: controllerAgentName}),
			NamespaceLister: listers.GetNamespaceLister(),
			Tracker:         GetTracker(ctx),
			Factory:         podspecable.Get(ctx),
		}
		return c
	}))
}
