/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestProvisionedService_GetGroupVersionKind(t *testing.T) {
	if got, want := (&ProvisionedService{}).GetGroupVersionKind().String(), "bindings.labs.vmware.com/v1alpha1, Kind=ProvisionedService"; got != want {
		t.Errorf("GetGroupVersionKind() = %v, want %v", got, want)
	}
}

func TestProvisionedService_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ProvisionedService
		expected *ProvisionedService
	}{
		{
			name:     "empty",
			seed:     &ProvisionedService{},
			expected: &ProvisionedService{},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			actual := c.seed.DeepCopy()
			actual.SetDefaults(context.TODO())
			if diff := cmp.Diff(c.expected, actual); diff != "" {
				t.Errorf("%s: SetDefaults() (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}

func TestProvisionedService_Validate(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ProvisionedService
		expected *apis.FieldError
	}{
		{
			name:     "empty",
			seed:     &ProvisionedService{},
			expected: apis.ErrMissingField("spec.binding.name"),
		},
		{
			name: "valid",
			seed: &ProvisionedService{
				Spec: ProvisionedServiceSpec{
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
			expected: nil,
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			actual := c.seed.Validate(context.TODO())
			if diff := cmp.Diff(c.expected.Error(), actual.Error()); diff != "" {
				t.Errorf("%s: Validate() (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}

func TestProvisionedService_GetStatus(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ProvisionedService
		expected *duckv1.Status
	}{
		{
			name:     "empty",
			seed:     &ProvisionedService{},
			expected: &duckv1.Status{},
		},
		{
			name: "status",
			seed: &ProvisionedService{
				Status: ProvisionedServiceStatus{
					Status: duckv1.Status{
						ObservedGeneration: 1,
						Conditions: duckv1.Conditions{
							{
								Type:   apis.ConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			expected: &duckv1.Status{
				ObservedGeneration: 1,
				Conditions: duckv1.Conditions{
					{
						Type:   apis.ConditionReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			actual := c.seed.GetStatus()
			if diff := cmp.Diff(c.expected, actual); diff != "" {
				t.Errorf("%s: GetStatus() (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}

func TestProvisionedService_GetConditionSet(t *testing.T) {
	expected := psCondSet
	actual := (&ProvisionedService{}).GetConditionSet()
	assert.Exactly(t, expected, actual)
}

func TestProvisionedServiceStatus_MarkReady(t *testing.T) {
	expected := &ProvisionedServiceStatus{
		Status: duckv1.Status{
			Conditions: duckv1.Conditions{
				{
					Type:   apis.ConditionReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	actual := &ProvisionedServiceStatus{}
	actual.MarkReady()

	if diff := cmp.Diff(expected, actual, cmpopts.IgnoreTypes(apis.VolatileTime{})); diff != "" {
		t.Errorf("MarkReady() (-expected, +actual): %s", diff)
	}
}

func TestProvisionedServiceStatus_InitializeConditions(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ProvisionedServiceStatus
		expected *ProvisionedServiceStatus
	}{
		{
			name: "empty",
			seed: &ProvisionedServiceStatus{},
			expected: &ProvisionedServiceStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   apis.ConditionReady,
							Status: corev1.ConditionUnknown,
						},
					},
				},
			},
		},
		{
			name: "preserve",
			seed: &ProvisionedServiceStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   apis.ConditionReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: &ProvisionedServiceStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   apis.ConditionReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			actual := c.seed.DeepCopy()
			actual.InitializeConditions()
			if diff := cmp.Diff(c.expected, actual, cmpopts.IgnoreTypes(apis.VolatileTime{})); diff != "" {
				t.Errorf("%s: InitializeConditions() (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}
