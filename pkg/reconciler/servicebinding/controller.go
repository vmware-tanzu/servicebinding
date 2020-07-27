/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package servicebinding

import (
	"context"

	servicev1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/service/v1alpha2"
	bindingclient "github.com/vmware-labs/service-bindings/pkg/client/injection/client"
	servicebindinginformer "github.com/vmware-labs/service-bindings/pkg/client/injection/informers/service/v1alpha2/servicebinding"
	servicebindingprojectioninformer "github.com/vmware-labs/service-bindings/pkg/client/injection/informers/serviceinternal/v1alpha2/servicebindingprojection"
	servicebindingreconciler "github.com/vmware-labs/service-bindings/pkg/client/injection/reconciler/service/v1alpha2/servicebinding"
	"github.com/vmware-labs/service-bindings/pkg/resolver"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	secretinformer "knative.dev/pkg/client/injection/kube/informers/core/v1/secret"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/tracker"
)

// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	logger := logging.FromContext(ctx)

	secretInformer := secretinformer.Get(ctx)
	serviceBindingProjectionInformer := servicebindingprojectioninformer.Get(ctx)
	serviceBindingInformer := servicebindinginformer.Get(ctx)

	r := &Reconciler{
		kubeclient:                     kubeclient.Get(ctx),
		bindingclient:                  bindingclient.Get(ctx),
		secretLister:                   secretInformer.Lister(),
		serviceBindingProjectionLister: serviceBindingProjectionInformer.Lister(),
	}
	impl := servicebindingreconciler.NewImpl(ctx, r)
	r.resolver = resolver.NewServiceableResolver(ctx, impl.EnqueueKey)

	logger.Info("Setting up event handlers.")

	serviceBindingInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	handleMatchingControllers := cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterControllerGK(servicev1alpha2.Kind("ServiceBinding")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	}
	secretInformer.Informer().AddEventHandler(handleMatchingControllers)
	serviceBindingProjectionInformer.Informer().AddEventHandler(handleMatchingControllers)

	r.tracker = tracker.New(impl.EnqueueKey, controller.GetTrackerLease(ctx))
	secretInformer.Informer().AddEventHandler(controller.HandleAll(
		controller.EnsureTypeMeta(
			r.tracker.OnChanged,
			corev1.SchemeGroupVersion.WithKind("Secret"),
		),
	))

	return impl
}
