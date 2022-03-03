/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package servicebinding

import (
	"context"

	servicebindingv1alpha3 "github.com/vmware-tanzu/servicebinding/pkg/apis/servicebinding/v1alpha3"
	bindingclient "github.com/vmware-tanzu/servicebinding/pkg/client/injection/client"
	servicebindingprojectioninformer "github.com/vmware-tanzu/servicebinding/pkg/client/injection/informers/labsinternal/v1alpha1/servicebindingprojection"
	servicebindinginformer "github.com/vmware-tanzu/servicebinding/pkg/client/injection/informers/servicebinding/v1alpha3/servicebinding"
	servicebindingreconciler "github.com/vmware-tanzu/servicebinding/pkg/client/injection/reconciler/servicebinding/v1alpha3/servicebinding"
	"github.com/vmware-tanzu/servicebinding/pkg/resolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
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

	serviceBindingProjectionInformer := servicebindingprojectioninformer.Get(ctx)
	serviceBindingInformer := servicebindinginformer.Get(ctx)

	r := &Reconciler{
		bindingclient:                  bindingclient.Get(ctx),
		serviceBindingProjectionLister: serviceBindingProjectionInformer.Lister(),
		now:                            metav1.Now,
	}
	impl := servicebindingreconciler.NewImpl(ctx, r)
	r.resolver = resolver.NewServiceableResolver(ctx, impl.EnqueueKey)

	logger.Info("Setting up event handlers.")

	serviceBindingInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	handleMatchingControllers := cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterControllerGK(servicebindingv1alpha3.Kind("ServiceBinding")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	}
	serviceBindingProjectionInformer.Informer().AddEventHandler(handleMatchingControllers)

	r.tracker = tracker.New(impl.EnqueueKey, controller.GetTrackerLease(ctx))

	return impl
}
