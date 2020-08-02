/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package servicebindingprojection

import (
	"context"

	servicebindinginternalv1alpha2listers "github.com/vmware-labs/service-bindings/pkg/client/listers/servicebindinginternal/v1alpha2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/webhook/psbinding"
)

var (
	_ controller.Reconciler  = (*ConditionalReconciler)(nil)
	_ reconciler.LeaderAware = (*ConditionalReconciler)(nil)
)

type delegateReconciler interface {
	controller.Reconciler
	reconciler.LeaderAware
}

type ConditionalReconciler struct {
	Delegate *psbinding.BaseReconciler
	Lister   servicebindinginternalv1alpha2listers.ServiceBindingProjectionLister
}

func (r *ConditionalReconciler) Reconcile(ctx context.Context, key string) error {
	logger := logging.FromContext(ctx)

	// convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	// ignore resources we are not a leader for
	if !r.Delegate.IsLeaderFor(types.NamespacedName{Namespace: namespace, Name: name}) {
		return nil
	}

	// get the resource with this namespace/name.
	original, err := r.Lister.ServiceBindingProjections(namespace).Get(name)

	if errors.IsNotFound(err) {
		// the resource may no longer exist, in which case we stop processing.
		logger.Debugf("resource %q no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	// don't modify the informers copy.
	resource := original.DeepCopy()

	isCustomProjection := resource.IsCustomProjection()
	hasOurFinalizer := sets.NewString(resource.GetFinalizers()...).Has(r.Delegate.GVR.GroupResource().String())
	if isCustomProjection && !hasOurFinalizer {
		// a third-party reconciler is responsible for this resource
		logger.Debugf("skipping reconciliation for %q, custom projection", key)
		return nil
	}

	// continue with our projection
	err = r.Delegate.Reconcile(ctx, key)
	if err != nil {
		return err
	}

	if isCustomProjection {
		// remove our finalizer since we are no longer responsible for this resource
		return r.Delegate.RemoveFinalizer(ctx, resource)
	}

	return nil
}

func (r *ConditionalReconciler) Promote(b reconciler.Bucket, enq func(reconciler.Bucket, types.NamespacedName)) error {
	return r.Delegate.Promote(b, enq)
}

func (r *ConditionalReconciler) Demote(b reconciler.Bucket) {
	r.Delegate.Demote(b)
}
