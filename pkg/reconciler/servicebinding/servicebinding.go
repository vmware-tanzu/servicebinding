/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package servicebinding

import (
	"context"
	"fmt"

	labsinternalv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labsinternal/v1alpha1"
	servicebindingv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha2"
	bindingclientset "github.com/vmware-labs/service-bindings/pkg/client/clientset/versioned"
	servicebindingreconciler "github.com/vmware-labs/service-bindings/pkg/client/injection/reconciler/servicebinding/v1alpha2/servicebinding"
	labsinternalv1alpha1listers "github.com/vmware-labs/service-bindings/pkg/client/listers/labsinternal/v1alpha1"
	"github.com/vmware-labs/service-bindings/pkg/reconciler/servicebinding/resources"
	resourcenames "github.com/vmware-labs/service-bindings/pkg/reconciler/servicebinding/resources/names"
	"github.com/vmware-labs/service-bindings/pkg/resolver"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/tracker"
)

// newReconciledNormal makes a new reconciler event with event type Normal, and
// reason ServiceBindingReconciled.
func newReconciledNormal(namespace, name string) reconciler.Event {
	return reconciler.NewEvent(corev1.EventTypeNormal, "Reconciled", "ServiceBinding reconciled: \"%s/%s\"", namespace, name)
}

// Reconciler implements servicebindingreconciler.Interface for
// ServiceBinding resources.
type Reconciler struct {
	bindingclient                  bindingclientset.Interface
	serviceBindingProjectionLister labsinternalv1alpha1listers.ServiceBindingProjectionLister

	resolver *resolver.ServiceableResolver
	tracker  tracker.Interface
}

// Check that our Reconciler implements Interface
var _ servicebindingreconciler.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, binding *servicebindingv1alpha2.ServiceBinding) reconciler.Event {
	logger := logging.FromContext(ctx)

	if binding.GetDeletionTimestamp() != nil {
		// Check for a DeletionTimestamp.  If present, elide the normal reconcile logic.
		// When a controller needs finalizer handling, it would go here.
		return nil
	}

	binding.Status.InitializeConditions()

	secretRef, err := r.provisionedSecret(ctx, logger, binding)
	if err != nil {
		return err
	}
	binding.Status.Binding = nil
	if secretRef != nil {
		binding.Status.Binding = &corev1.LocalObjectReference{
			Name: secretRef.Name,
		}
		binding.Status.MarkServiceAvailable()
	}

	serviceBindingProjection, err := r.serviceBindingProjection(ctx, logger, binding)
	if err != nil {
		return err
	}
	if serviceBindingProjection != nil {
		binding.Status.PropagateServiceBindingProjectionStatus(serviceBindingProjection)
	}

	binding.Status.ObservedGeneration = binding.Generation

	return newReconciledNormal(binding.Namespace, binding.Name)
}

func (r *Reconciler) provisionedSecret(ctx context.Context, logger *zap.SugaredLogger, binding *servicebindingv1alpha2.ServiceBinding) (*corev1.LocalObjectReference, error) {
	serviceRef := binding.Spec.Service.DeepCopy()
	serviceRef.Namespace = binding.Namespace
	return r.resolver.ServiceableFromObjectReference(ctx, serviceRef, binding)
}

func (r *Reconciler) serviceBindingProjection(ctx context.Context, logger *zap.SugaredLogger, binding *servicebindingv1alpha2.ServiceBinding) (*labsinternalv1alpha1.ServiceBindingProjection, error) {
	recorder := controller.GetEventRecorder(ctx)

	if binding.Status.Binding == nil {
		return nil, nil
	}

	serviceBindingProjectionName := resourcenames.ServiceBindingProjection(binding)
	serviceBindingProjection, err := r.serviceBindingProjectionLister.ServiceBindingProjections(binding.Namespace).Get(serviceBindingProjectionName)
	if apierrs.IsNotFound(err) {
		serviceBindingProjection, err = r.createServiceBindingProjection(ctx, binding)
		if err != nil {
			recorder.Eventf(binding, corev1.EventTypeWarning, "CreationFailed", "Failed to create ServiceBindingProjection %q: %v", serviceBindingProjectionName, err)
			return nil, fmt.Errorf("failed to create ServiceBindingProjection: %w", err)
		}
		recorder.Eventf(binding, corev1.EventTypeNormal, "Created", "Created ServiceBindingProjection %q", serviceBindingProjectionName)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get ServiceBindingProjection: %w", err)
	} else if !metav1.IsControlledBy(serviceBindingProjection, binding) {
		return nil, fmt.Errorf("ServiceBinding %q does not own ServiceBindingProjection: %q", binding.Name, serviceBindingProjectionName)
	} else if serviceBindingProjection, err = r.reconcileServiceBindingProjection(ctx, binding, serviceBindingProjection); err != nil {
		return nil, fmt.Errorf("failed to reconcile ServiceBindingProjection: %w", err)
	}
	return serviceBindingProjection, nil
}

func (c *Reconciler) createServiceBindingProjection(ctx context.Context, binding *servicebindingv1alpha2.ServiceBinding) (*labsinternalv1alpha1.ServiceBindingProjection, error) {
	serviceBindingProjection, err := resources.MakeServiceBindingProjection(binding)
	if err != nil {
		return nil, err
	}
	return c.bindingclient.InternalV1alpha1().ServiceBindingProjections(binding.Namespace).Create(ctx, serviceBindingProjection, metav1.CreateOptions{})
}

func serviceBindingProjectionSemanticEquals(ctx context.Context, desiredServiceBindingProjection, serviceBindingProjection *labsinternalv1alpha1.ServiceBindingProjection) (bool, error) {
	return equality.Semantic.DeepEqual(desiredServiceBindingProjection.Spec, serviceBindingProjection.Spec) &&
		equality.Semantic.DeepEqual(desiredServiceBindingProjection.ObjectMeta.Labels, serviceBindingProjection.ObjectMeta.Labels) &&
		equality.Semantic.DeepEqual(desiredServiceBindingProjection.ObjectMeta.Annotations, serviceBindingProjection.ObjectMeta.Annotations), nil
}

func (c *Reconciler) reconcileServiceBindingProjection(ctx context.Context, binding *servicebindingv1alpha2.ServiceBinding, projection *labsinternalv1alpha1.ServiceBindingProjection) (*labsinternalv1alpha1.ServiceBindingProjection, error) {
	existing := projection.DeepCopy()
	// In the case of an upgrade, there can be default values set that don't exist pre-upgrade.
	// We are setting the up-to-date default values here so an update won't be triggered if the only
	// diff is the new default values.
	desired, err := resources.MakeServiceBindingProjection(binding)
	if err != nil {
		return nil, err
	}

	if equals, err := serviceBindingProjectionSemanticEquals(ctx, desired, existing); err != nil {
		return nil, err
	} else if equals {
		return projection, nil
	}

	// Preserve the rest of the object (e.g. ObjectMeta except for labels).
	existing.Spec = desired.Spec
	existing.ObjectMeta.Labels = desired.ObjectMeta.Labels
	existing.ObjectMeta.Annotations = desired.ObjectMeta.Annotations
	return c.bindingclient.InternalV1alpha1().ServiceBindingProjections(binding.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
}
