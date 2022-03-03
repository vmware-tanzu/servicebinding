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

	provisionedserviceinformer "github.com/vmware-tanzu/servicebinding/pkg/client/injection/informers/labs/v1alpha1/provisionedservice"
	provisionedservicereconciler "github.com/vmware-tanzu/servicebinding/pkg/client/injection/reconciler/labs/v1alpha1/provisionedservice"
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
