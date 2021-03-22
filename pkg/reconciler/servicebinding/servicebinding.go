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
	return reconciler.NewEvent(corev1.EventTypeNormal, "Reconciled", "ServiceBinding reconciled: \"%s/%s\"", namespace, name)
}

// Reconciler implements servicebindingreconciler.Interface for
// ServiceBinding resources.
type Reconciler struct {
	kubeclient                     kubernetes.Interface
	bindingclient                  bindingclientset.Interface
	secretLister                   corev1listers.SecretLister
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

	secret, err := r.projectedSecret(ctx, logger, binding)
	if err != nil {
		return err
	}
	binding.Status.Binding = nil
	if secret != nil {
		binding.Status.Binding = &corev1.LocalObjectReference{
			Name: secret.Name,
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

func (r *Reconciler) projectedSecret(ctx context.Context, logger *zap.SugaredLogger, binding *servicebindingv1alpha2.ServiceBinding) (*corev1.Secret, error) {
	recorder := controller.GetEventRecorder(ctx)

	serviceRef := binding.Spec.Service.DeepCopy()
	serviceRef.Namespace = binding.Namespace
	providerRef, err := r.resolver.ServiceableFromObjectReference(ctx, serviceRef, binding)
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
		projection, err = r.createProjectedSecret(ctx, binding, reference)
		if err != nil {
			recorder.Eventf(binding, corev1.EventTypeWarning, "CreationFailed", "Failed to create projected Secret %q: %v", projectionName, err)
			return nil, fmt.Errorf("failed to create projected Secret: %w", err)
		}
		recorder.Eventf(binding, corev1.EventTypeNormal, "Created", "Created projected Secret %q", projectionName)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get projected Secret: %w", err)
	} else if !metav1.IsControlledBy(projection, binding) {
		return nil, fmt.Errorf("ServiceBinding %q does not own projected Secret: %q", binding.Name, projectionName)
	} else if projection, err = r.reconcileProtectedSecret(ctx, binding, reference, projection); err != nil {
		return nil, fmt.Errorf("failed to reconcile projected Secret: %w", err)
	}
	return projection, nil
}

func (c *Reconciler) createProjectedSecret(ctx context.Context, binding *servicebindingv1alpha2.ServiceBinding, reference *corev1.Secret) (*corev1.Secret, error) {
	projection, err := resources.MakeProjectedSecret(binding, reference)
	if err != nil {
		return nil, err
	}
	return c.kubeclient.CoreV1().Secrets(binding.Namespace).Create(ctx, projection, metav1.CreateOptions{})
}

func projectedSecretSemanticEquals(ctx context.Context, desiredProjection, projection *corev1.Secret) (bool, error) {
	return equality.Semantic.DeepEqual(desiredProjection.Type, projection.Type) &&
		equality.Semantic.DeepEqual(desiredProjection.Data, projection.Data) &&
		equality.Semantic.DeepEqual(desiredProjection.ObjectMeta.Labels, projection.ObjectMeta.Labels) &&
		equality.Semantic.DeepEqual(desiredProjection.ObjectMeta.Annotations, projection.ObjectMeta.Annotations), nil
}

func (c *Reconciler) reconcileProtectedSecret(ctx context.Context, binding *servicebindingv1alpha2.ServiceBinding, reference, projection *corev1.Secret) (*corev1.Secret, error) {
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

	// Preserve the rest of the object (e.g. ObjectMeta except for labels and annotations).
	existing.Data = desiredProjection.Data
	existing.Type = desiredProjection.Type
	existing.ObjectMeta.Annotations = desiredProjection.ObjectMeta.Annotations
	existing.ObjectMeta.Labels = desiredProjection.ObjectMeta.Labels
	return c.kubeclient.CoreV1().Secrets(binding.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
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
