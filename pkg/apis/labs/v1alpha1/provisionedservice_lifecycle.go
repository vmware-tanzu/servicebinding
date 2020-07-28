/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

const (
	ProvisionedServiceConditionReady = apis.ConditionReady
)

var psCondSet = apis.NewLivingConditionSet()

func (p *ProvisionedService) GetStatus() *duckv1.Status {
	return &p.Status.Status
}

func (p *ProvisionedService) GetConditionSet() apis.ConditionSet {
	return psCondSet
}

func (ps *ProvisionedServiceStatus) MarkReady() {
	psCondSet.Manage(ps).MarkTrue(ProvisionedServiceConditionReady)
}

func (ps *ProvisionedServiceStatus) InitializeConditions() {
	psCondSet.Manage(ps).InitializeConditions()
}
