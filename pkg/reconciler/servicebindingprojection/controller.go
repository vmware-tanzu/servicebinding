/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package servicebindingprojection

import (
	"context"

	labsinternalv1alpha1 "github.com/vmware-tanzu/servicebinding/pkg/apis/labsinternal/v1alpha1"
	servicebindingprojectioninformer "github.com/vmware-tanzu/servicebinding/pkg/client/injection/informers/labsinternal/v1alpha1/servicebindingprojection"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis/duck"
	"knative.dev/pkg/client/injection/ducks/duck/v1/podspecable"
	nsinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/namespace"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/clients/dynamicclient"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/tracker"
	"knative.dev/pkg/webhook/psbinding"
)

const (
	controllerAgentName = "servicebindingprojection-controller"
)

// NewController returns a new ServiceBindingProjection reconciler.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	logger := logging.FromContext(ctx)
	serviceBindingProjectionInformer := servicebindingprojectioninformer.Get(ctx)
	nsInformer := nsinformer.Get(ctx)
	dc := dynamicclient.Get(ctx)

	psInformerFactory := podspecable.Get(ctx)
	c := &psbinding.BaseReconciler{
		GVR: labsinternalv1alpha1.SchemeGroupVersion.WithResource("servicebindingprojections"),
		Get: func(namespace string, name string) (psbinding.Bindable, error) {
			return serviceBindingProjectionInformer.Lister().ServiceBindingProjections(namespace).Get(name)
		},
		DynamicClient: dc,
		Recorder: record.NewBroadcaster().NewRecorder(
			scheme.Scheme, corev1.EventSource{Component: controllerAgentName}),
		NamespaceLister: nsInformer.Lister(),
	}

	impl := controller.NewImpl(c, logger, "ServiceBindingProjections")

	logger.Info("Setting up event handlers")

	serviceBindingProjectionInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	c.Tracker = tracker.New(impl.EnqueueKey, controller.GetTrackerLease(ctx))
	c.Factory = &duck.CachedInformerFactory{
		Delegate: &duck.EnqueueInformerFactory{
			Delegate:     psInformerFactory,
			EventHandler: controller.HandleAll(c.Tracker.OnChanged),
		},
	}
	return impl
}

func ListAll(ctx context.Context, handler cache.ResourceEventHandler) psbinding.ListAll {
	serviceBindingProjectionInformer := servicebindingprojectioninformer.Get(ctx)

	// Whenever a ServiceBindingProjection changes our webhook programming might change.
	serviceBindingProjectionInformer.Informer().AddEventHandler(handler)

	return func() ([]psbinding.Bindable, error) {
		l, err := serviceBindingProjectionInformer.Lister().List(labels.Everything())
		if err != nil {
			return nil, err
		}
		bl := make([]psbinding.Bindable, 0, len(l))
		for _, elt := range l {
			bl = append(bl, elt)
		}
		return bl, nil
	}
}
