/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha3

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"

	labsinternalv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labsinternal/v1alpha1"
)

const (
	ServiceBindingConditionReady            = "Ready"
	ServiceBindingConditionServiceAvailable = "ServiceAvailable"
	ServiceBindingConditionProjectionReady  = "ProjectionReady"
	InitializeConditionReason               = "Unknown"
)

func (bs *ServiceBindingStatus) InitializeConditions(now metav1.Time) {
	ready := metav1.Condition{Type: ServiceBindingConditionReady, Status: metav1.ConditionUnknown, LastTransitionTime: now, Reason: InitializeConditionReason}
	serviceAvailable := metav1.Condition{Type: ServiceBindingConditionServiceAvailable, Status: metav1.ConditionUnknown, LastTransitionTime: now, Reason: InitializeConditionReason}
	projectionReady := metav1.Condition{Type: ServiceBindingConditionProjectionReady, Status: metav1.ConditionUnknown, LastTransitionTime: now, Reason: InitializeConditionReason}
	for _, c := range bs.Conditions {
		switch c.Type {
		case ServiceBindingConditionReady:
			ready = c
		case ServiceBindingConditionServiceAvailable:
			serviceAvailable = c
		case ServiceBindingConditionProjectionReady:
			projectionReady = c
		}
	}
	bs.Conditions = []metav1.Condition{ready, serviceAvailable, projectionReady}
}

func (bs *ServiceBindingStatus) MarkServiceAvailable(now metav1.Time) {
	if bs.Conditions[1].Status != metav1.ConditionTrue {
		bs.Conditions[1].LastTransitionTime = now
	}
	bs.Conditions[1].Status = metav1.ConditionTrue
	bs.Conditions[1].Reason = "Available"
	bs.Conditions[1].Message = ""

	bs.aggregateReadyCondition(now)
}

func (bs *ServiceBindingStatus) MarkServiceUnavailable(reason string, message string, now metav1.Time) {
	if bs.Conditions[1].Status != metav1.ConditionFalse {
		bs.Conditions[1].LastTransitionTime = now
	}
	bs.Conditions[1].Status = metav1.ConditionFalse
	bs.Conditions[1].Reason = reason
	bs.Conditions[1].Message = message

	bs.aggregateReadyCondition(now)
}

func (bs *ServiceBindingStatus) PropagateServiceBindingProjectionStatus(bp *labsinternalv1alpha1.ServiceBindingProjection, now metav1.Time) {
	if bp == nil {
		return
	}
	sbpready := bp.Status.GetCondition(labsinternalv1alpha1.ServiceBindingProjectionConditionReady)
	if sbpready == nil {
		sbpready = &apis.Condition{}
	}

	newStatus := metav1.ConditionStatus(sbpready.Status)
	if newStatus == "" {
		newStatus = metav1.ConditionUnknown
	}

	if bs.Conditions[2].Status != newStatus {
		bs.Conditions[2].LastTransitionTime = now
	}
	bs.Conditions[2].Status = newStatus
	if sbpready.Reason != "" {
		bs.Conditions[2].Reason = sbpready.Reason
	} else if bs.Conditions[2].Status == metav1.ConditionTrue {
		bs.Conditions[2].Reason = "Projected"
	} else {
		bs.Conditions[2].Reason = "Unknown"
	}
	bs.Conditions[2].Message = sbpready.Message

	bs.aggregateReadyCondition(now)
}

func (bs *ServiceBindingStatus) aggregateReadyCondition(now metav1.Time) {
	currentStatus := bs.Conditions[0].Status
	if bs.Conditions[1].Status == metav1.ConditionTrue && bs.Conditions[2].Status == metav1.ConditionTrue {
		bs.Conditions[0].Status = metav1.ConditionTrue
		bs.Conditions[0].Reason = "Ready"
		bs.Conditions[0].Message = ""
	} else if bs.Conditions[1].Status == metav1.ConditionFalse {
		bs.Conditions[0].Status = metav1.ConditionFalse
		bs.Conditions[0].Reason = fmt.Sprintf("%s%s", ServiceBindingConditionServiceAvailable, bs.Conditions[1].Reason)
		bs.Conditions[0].Message = bs.Conditions[1].Message
	} else if bs.Conditions[2].Status == metav1.ConditionFalse {
		bs.Conditions[0].Status = metav1.ConditionFalse
		bs.Conditions[0].Reason = fmt.Sprintf("%s%s", ServiceBindingConditionProjectionReady, bs.Conditions[2].Reason)
		bs.Conditions[0].Message = bs.Conditions[2].Message
	} else if bs.Conditions[1].Status == metav1.ConditionUnknown {
		bs.Conditions[0].Status = metav1.ConditionUnknown
		bs.Conditions[0].Reason = fmt.Sprintf("%s%s", ServiceBindingConditionServiceAvailable, bs.Conditions[1].Reason)
		bs.Conditions[0].Message = bs.Conditions[1].Message
	} else if bs.Conditions[2].Status == metav1.ConditionUnknown {
		bs.Conditions[0].Status = metav1.ConditionUnknown
		bs.Conditions[0].Reason = fmt.Sprintf("%s%s", ServiceBindingConditionProjectionReady, bs.Conditions[2].Reason)
		bs.Conditions[0].Message = bs.Conditions[2].Message
	} else {
		bs.Conditions[0].Status = metav1.ConditionUnknown
		bs.Conditions[0].Reason = "Unknown"
		bs.Conditions[0].Message = ""
	}

	// update time when the status changes
	if bs.Conditions[0].Status != currentStatus {
		bs.Conditions[0].LastTransitionTime = now
	}
}

// required for knative, not used
func (b *ServiceBinding) GetStatus() *duckv1.Status {
	return &duckv1.Status{}
}

// required for knative, not used
func (b *ServiceBinding) GetConditionSet() apis.ConditionSet {
	return apis.NewLivingConditionSet()
}
