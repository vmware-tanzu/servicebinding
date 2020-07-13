/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package provisionedservice

import (
	"context"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"

	provisionedserviceinformer "github.com/vmware-labs/service-bindings/pkg/client/injection/informers/bindings/v1alpha1/provisionedservice"
	provisionedservicereconciler "github.com/vmware-labs/service-bindings/pkg/client/injection/reconciler/bindings/v1alpha1/provisionedservice"
)

// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	logger := logging.FromContext(ctx)

	provisionedserviceInformer := provisionedserviceinformer.Get(ctx)

	r := &Reconciler{}
	impl := provisionedservicereconciler.NewImpl(ctx, r)

	logger.Info("Setting up event handlers.")

	provisionedserviceInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	return impl
}
