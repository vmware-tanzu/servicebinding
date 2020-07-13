/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package provisionedservice

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	bindingsv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/bindings/v1alpha1"
	provisionedservicereconciler "github.com/vmware-labs/service-bindings/pkg/client/injection/reconciler/bindings/v1alpha1/provisionedservice"
	"knative.dev/pkg/reconciler"
)

// newReconciledNormal makes a new reconciler event with event type Normal, and
// reason ProvisionedServiceReconciled.
func newReconciledNormal(namespace, name string) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeNormal, "ProvisionedServiceReconciled", "ProvisionedService reconciled: \"%s/%s\"", namespace, name)
}

// Reconciler implements provisionedservicereconciler.Interface for
// ProvisionedService resources.
type Reconciler struct{}

// Check that our Reconciler implements Interface
var _ provisionedservicereconciler.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, o *bindingsv1alpha1.ProvisionedService) reconciler.Event {
	if o.GetDeletionTimestamp() != nil {
		// Check for a DeletionTimestamp.  If present, elide the normal reconcile logic.
		// When a controller needs finalizer handling, it would go here.
		return nil
	}
	o.Status.InitializeConditions()

	o.Status.Binding = o.Spec.Binding

	o.Status.ObservedGeneration = o.Generation
	o.Status.MarkReady()
	return newReconciledNormal(o.Namespace, o.Name)
}
