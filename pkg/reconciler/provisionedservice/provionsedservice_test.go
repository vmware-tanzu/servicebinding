/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package provisionedservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	labsv1alpha1 "github.com/vmware-tanzu/servicebinding/pkg/apis/labs/v1alpha1"
	servicebindingsclient "github.com/vmware-tanzu/servicebinding/pkg/client/injection/client/fake"
	provisionedservicereconciler "github.com/vmware-tanzu/servicebinding/pkg/client/injection/reconciler/labs/v1alpha1/provisionedservice"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgotesting "k8s.io/client-go/testing"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"

	// register injection fakes
	_ "github.com/vmware-tanzu/servicebinding/pkg/client/injection/informers/labs/v1alpha1/provisionedservice/fake"

	. "github.com/vmware-tanzu/servicebinding/pkg/reconciler/testing"
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
	now := metav1.NewTime(time.Now())
	binding := corev1.LocalObjectReference{
		Name: "my-binding",
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
			&labsv1alpha1.ProvisionedService{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:         namespace,
					Name:              name,
					DeletionTimestamp: &now,
				},
				Spec: labsv1alpha1.ProvisionedServiceSpec{
					Binding: binding,
				},
			},
		},
	}, {
		Name: "nop - in sync",
		Key:  key,
		Objects: []runtime.Object{
			&labsv1alpha1.ProvisionedService{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: labsv1alpha1.ProvisionedServiceSpec{
					Binding: binding,
				},
				Status: labsv1alpha1.ProvisionedServiceStatus{
					Binding: binding,
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   labsv1alpha1.ProvisionedServiceConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		},
		WantEvents: []string{
			Eventf(corev1.EventTypeNormal, "Reconciled", "ProvisionedService reconciled: %q", key),
		},
	}, {
		Name: "reflect binding on status",
		Key:  key,
		Objects: []runtime.Object{
			&labsv1alpha1.ProvisionedService{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: labsv1alpha1.ProvisionedServiceSpec{
					Binding: binding,
				},
			},
		},
		WantStatusUpdates: []clientgotesting.UpdateActionImpl{{
			Object: &labsv1alpha1.ProvisionedService{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:  namespace,
					Name:       name,
					Generation: 1,
				},
				Spec: labsv1alpha1.ProvisionedServiceSpec{
					Binding: binding,
				},
				Status: labsv1alpha1.ProvisionedServiceStatus{
					Binding: binding,
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   labsv1alpha1.ProvisionedServiceConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
		}},
		WantEvents: []string{
			Eventf(corev1.EventTypeNormal, "Reconciled", "ProvisionedService reconciled: %q", key),
		},
	}}

	table.Test(t, MakeFactory(func(ctx context.Context, listers *Listers, cmw configmap.Watcher) controller.Reconciler {
		r := &Reconciler{}

		return provisionedservicereconciler.NewReconciler(ctx, logging.FromContext(ctx), servicebindingsclient.Get(ctx),
			listers.GetProvisionedServiceLister(), controller.GetEventRecorder(ctx), r)
	}))
}
