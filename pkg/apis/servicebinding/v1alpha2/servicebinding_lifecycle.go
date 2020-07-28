/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	servicebindinginternalv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebindinginternal/v1alpha2"
)

const (
	ServiceBindingConditionReady            = apis.ConditionReady
	ServiceBindingConditionProjectionReady  = "ProjectionReady"
	ServiceBindingConditionServiceAvailable = "ServiceAvailable"
)

var sbCondSet = apis.NewLivingConditionSet(
	ServiceBindingConditionProjectionReady,
	ServiceBindingConditionServiceAvailable,
)

func (b *ServiceBinding) GetStatus() *duckv1.Status {
	return &b.Status.Status
}

func (b *ServiceBinding) GetConditionSet() apis.ConditionSet {
	return sbCondSet
}

func (bs *ServiceBindingStatus) InitializeConditions() {
	sbCondSet.Manage(bs).InitializeConditions()
}

func (bs *ServiceBindingStatus) PropagateServiceBindingProjectionStatus(bp *servicebindinginternalv1alpha2.ServiceBindingProjection) {
	if bp == nil {
		return
	}
	sbpready := bp.Status.GetCondition(servicebindinginternalv1alpha2.ServiceBindingProjectionConditionReady)
	if sbpready == nil {
		return
	}
	switch sbpready.Status {
	case corev1.ConditionTrue:
		sbCondSet.Manage(bs).MarkTrueWithReason(ServiceBindingConditionProjectionReady, sbpready.Reason, sbpready.Message)
	case corev1.ConditionFalse:
		sbCondSet.Manage(bs).MarkFalse(ServiceBindingConditionProjectionReady, sbpready.Reason, sbpready.Message)
	default:
		sbCondSet.Manage(bs).MarkUnknown(ServiceBindingConditionProjectionReady, sbpready.Reason, sbpready.Message)
	}
}

func (bs *ServiceBindingStatus) MarkServiceAvailable() {
	sbCondSet.Manage(bs).MarkTrue(ServiceBindingConditionServiceAvailable)
}

func (bs *ServiceBindingStatus) MarkServiceUnavailable(reason string, message string) {
	sbCondSet.Manage(bs).MarkFalse(
		ServiceBindingConditionServiceAvailable, reason, message)
}
