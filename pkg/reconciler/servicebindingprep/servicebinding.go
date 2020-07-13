/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package servicebindingprep

import (
	"context"
	"fmt"

	servicev1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/service/v1alpha2"
	servicebindingreconciler "github.com/vmware-labs/service-bindings/pkg/client/injection/reconciler/service/v1alpha2/servicebinding"
	"github.com/vmware-labs/service-bindings/pkg/reconciler/servicebindingprep/resources"
	resourcenames "github.com/vmware-labs/service-bindings/pkg/reconciler/servicebindingprep/resources/names"
	"github.com/vmware-labs/service-bindings/pkg/resolver"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/tracker"
)

// newReconciledNormal makes a new reconciler event with event type Normal, and
// reason ServiceBindingReconciled.
func newReconciledNormal(namespace, name string) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeNormal, "ServiceBindingReconciled", "ServiceBinding reconciled: \"%s/%s\"", namespace, name)
}

// Reconciler implements servicebindingreconciler.Interface for
// ServiceBinding resources.
type Reconciler struct {
	kubeclient   kubernetes.Interface
	secretLister corev1listers.SecretLister

	resolver *resolver.ServiceableResolver
	tracker  tracker.Interface
}

// Check that our Reconciler implements Interface
var _ servicebindingreconciler.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, binding *servicev1alpha2.ServiceBinding) reconciler.Event {
	logger := logging.FromContext(ctx)

	if binding.GetDeletionTimestamp() != nil {
		// Check for a DeletionTimestamp.  If present, elide the normal reconcile logic.
		// When a controller needs finalizer handling, it would go here.
		return nil
	}
	// let the main controller manage the common status fields
	// binding.Status.InitializeConditions()

	projection, err := r.projectedSecret(ctx, logger, binding)
	if err != nil {
		return err
	}
	binding.Status.Binding = nil
	if projection != nil {
		binding.Status.Binding = &corev1.LocalObjectReference{
			Name: projection.Name,
		}
	}

	// let the main controller manage the common status fields
	// binding.Status.ObservedGeneration = binding.Generation
	// binding.Status.MarkReady()
	return newReconciledNormal(binding.Namespace, binding.Name)
}

func (r *Reconciler) projectedSecret(ctx context.Context, logger *zap.SugaredLogger, binding *servicev1alpha2.ServiceBinding) (*corev1.Secret, error) {
	recorder := controller.GetEventRecorder(ctx)

	providerRef, err := r.resolver.ServiceableFromObjectReference(binding.Spec.Service, binding)
	if err != nil {
		return nil, err
	}
	r.tracker.TrackReference(tracker.Reference{
		APIVersion: "v1",
		Kind:       "Secret",
		Namespace:  binding.Namespace,
		Name:       providerRef.Name,
	}, binding)
	reference, err := r.secretLister.Secrets(binding.Namespace).Get(providerRef.Name)
	if apierrs.IsNotFound(err) {
		return nil, nil
	}

	projectionName := resourcenames.ProjectedSecret(binding)
	projection, err := r.secretLister.Secrets(binding.Namespace).Get(projectionName)
	if apierrs.IsNotFound(err) {
		projection, err = r.createProjectedSecret(binding, reference)
		if err != nil {
			recorder.Eventf(binding, corev1.EventTypeWarning, "CreationFailed", "Failed to create projected Secret %q: %v", projectionName, err)
			return nil, fmt.Errorf("failed to create projected Secret: %w", err)
		}
		recorder.Eventf(binding, corev1.EventTypeNormal, "Created", "Created projected Secret %q", projectionName)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get projected Secret: %w", err)
	} else if !metav1.IsControlledBy(projection, binding) {
		return nil, fmt.Errorf("service: %q does not own projected secret: %q", binding.Name, projectionName)
	} else if projection, err = r.reconcileProtectedSecret(ctx, binding, reference, projection); err != nil {
		return nil, fmt.Errorf("failed to reconcile projected Secret: %w", err)
	}
	return projection, nil
}

func (c *Reconciler) createProjectedSecret(binding *servicev1alpha2.ServiceBinding, reference *corev1.Secret) (*corev1.Secret, error) {
	projection, err := resources.MakeProjectedSecret(binding, reference)
	if err != nil {
		return nil, err
	}
	return c.kubeclient.CoreV1().Secrets(binding.Namespace).Create(projection)
}

func projectedSecretSemanticEquals(ctx context.Context, desiredProjection, projection *corev1.Secret) (bool, error) {
	return equality.Semantic.DeepEqual(desiredProjection.Data, projection.Data) &&
		equality.Semantic.DeepEqual(desiredProjection.ObjectMeta.Labels, projection.ObjectMeta.Labels) &&
		equality.Semantic.DeepEqual(desiredProjection.ObjectMeta.Annotations, projection.ObjectMeta.Annotations), nil
}

func (c *Reconciler) reconcileProtectedSecret(ctx context.Context, binding *servicev1alpha2.ServiceBinding, reference, projection *corev1.Secret) (*corev1.Secret, error) {
	existing := projection.DeepCopy()
	// In the case of an upgrade, there can be default values set that don't exist pre-upgrade.
	// We are setting the up-to-date default values here so an update won't be triggered if the only
	// diff is the new default values.
	desiredProjection, err := resources.MakeProjectedSecret(binding, reference)
	if err != nil {
		return nil, err
	}

	if equals, err := projectedSecretSemanticEquals(ctx, desiredProjection, existing); err != nil {
		return nil, err
	} else if equals {
		return projection, nil
	}

	// Preserve the rest of the object (e.g. ObjectMeta except for labels).
	existing.Data = desiredProjection.Data
	existing.ObjectMeta.Labels = desiredProjection.ObjectMeta.Labels
	return c.kubeclient.CoreV1().Secrets(binding.Namespace).Update(existing)
}
