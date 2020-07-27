/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	serviceinternalv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/serviceinternal/v1alpha2"
)

const (
	ServiceBindingConditionReady            = apis.ConditionReady
	ServiceBindingConditionProjectionReady  = "ProjectionReady"
	ServiceBindingConditionServiceAvailable = "ServiceAvailable"
)

var serviceCondSet = apis.NewLivingConditionSet(
	ServiceBindingConditionProjectionReady,
	ServiceBindingConditionServiceAvailable,
)

func (b *ServiceBinding) GetStatus() *duckv1.Status {
	return &b.Status.Status
}

func (b *ServiceBinding) GetConditionSet() apis.ConditionSet {
	return serviceCondSet
}

func (bs *ServiceBindingStatus) InitializeConditions() {
	serviceCondSet.Manage(bs).InitializeConditions()
}

func (bs *ServiceBindingStatus) PropagateServiceBindingProjectionStatus(bp *serviceinternalv1alpha2.ServiceBindingProjection) {
	if bp == nil {
		return
	}
	sbpready := bp.Status.GetCondition(serviceinternalv1alpha2.ServiceBindingConditionReady)
	if sbpready == nil {
		return
	}
	switch sbpready.Status {
	case corev1.ConditionTrue:
		serviceCondSet.Manage(bs).MarkTrueWithReason(ServiceBindingConditionProjectionReady, sbpready.Reason, sbpready.Message)
	case corev1.ConditionFalse:
		serviceCondSet.Manage(bs).MarkFalse(ServiceBindingConditionProjectionReady, sbpready.Reason, sbpready.Message)
	default:
		serviceCondSet.Manage(bs).MarkUnknown(ServiceBindingConditionProjectionReady, sbpready.Reason, sbpready.Message)
	}
}

func (bs *ServiceBindingStatus) MarkServiceAvailable() {
	serviceCondSet.Manage(bs).MarkTrue(ServiceBindingConditionServiceAvailable)
}

func (bs *ServiceBindingStatus) MarkServiceUnavailable(reason string, message string) {
	serviceCondSet.Manage(bs).MarkFalse(
		ServiceBindingConditionServiceAvailable, reason, message)
}

func (bs *ServiceBindingStatus) SetObservedGeneration(gen int64) {
	bs.ObservedGeneration = gen
}
