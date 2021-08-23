/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha2

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	labsinternalv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labsinternal/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/tracker"
)

func TestServiceBinding_GetGroupVersionKind(t *testing.T) {
	if got, want := (&ServiceBinding{}).GetGroupVersionKind().String(), "service.binding/v1alpha2, Kind=ServiceBinding"; got != want {
		t.Errorf("GetGroupVersionKind() = %v, want %v", got, want)
	}
}

func TestServiceBinding_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ServiceBinding
		expected *ServiceBinding
	}{
		{
			name: "empty",
			seed: &ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
			},
			expected: &ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingSpec{
					Name: "my-binding",
				},
			},
		},
		{
			name: "custom name",
			seed: &ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingSpec{
					Name: "custom-binding",
				},
			},
			expected: &ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingSpec{
					Name: "custom-binding",
				},
			},
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

func TestServiceBinding_Validate(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ServiceBinding
		expected *apis.FieldError
	}{
		{
			name: "empty",
			seed: &ServiceBinding{},
			expected: (&apis.FieldError{}).Also(
				apis.ErrMissingField("spec.workload"),
				apis.ErrMissingField("spec.service"),
			),
		},
		{
			name: "valid",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Workload: &WorkloadReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-workload",
						},
					},
					Service: &tracker.Reference{
						APIVersion: "bindings.labs.vmware.com/v1alpha1",
						Kind:       "ProvisionedService",
						Name:       "my-service",
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid, workload selector",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Workload: &WorkloadReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Selector:   &metav1.LabelSelector{},
						},
					},
					Service: &tracker.Reference{
						APIVersion: "bindings.labs.vmware.com/v1alpha1",
						Kind:       "ProvisionedService",
						Name:       "my-service",
					},
				},
			},
			expected: nil,
		},
		{
			name: "invalid, service selector",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Workload: &WorkloadReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-workload",
						},
					},
					Service: &tracker.Reference{
						APIVersion: "bindings.labs.vmware.com/v1alpha1",
						Kind:       "ProvisionedService",
						Selector:   &metav1.LabelSelector{},
					},
				},
			},
			expected: (&apis.FieldError{}).Also(
				apis.ErrMissingField("spec.service.name"),
			),
		},
		{
			name: "valid, env",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Workload: &WorkloadReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
					},
					Service: &tracker.Reference{
						APIVersion: "bindings.labs.vmware.com/v1alpha1",
						Kind:       "ProvisionedService",
						Name:       "my-service",
					},
					Env: []EnvVar{
						{
							Name: "MY_VAR",
							Key:  "my-key",
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "disallow namespaces",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Workload: &WorkloadReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
							Namespace:  "default",
						},
					},
					Service: &tracker.Reference{
						APIVersion: "bindings.labs.vmware.com/v1alpha1",
						Kind:       "ProvisionedService",
						Name:       "my-service",
						Namespace:  "default",
					},
				},
			},
			expected: (&apis.FieldError{}).Also(
				apis.ErrDisallowedFields("spec.workload.namespace"),
				apis.ErrDisallowedFields("spec.service.namespace"),
			),
		},
		{
			name: "empty env",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Workload: &WorkloadReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
					},
					Service: &tracker.Reference{
						APIVersion: "bindings.labs.vmware.com/v1alpha1",
						Kind:       "ProvisionedService",
						Name:       "my-service",
					},
					Env: []EnvVar{
						{},
					},
				},
			},
			expected: (&apis.FieldError{}).Also(
				apis.ErrMissingField("spec.env[0].name"),
				apis.ErrMissingField("spec.env[0].key"),
			),
		},
		{
			name: "duplicate env",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Workload: &WorkloadReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
					},
					Service: &tracker.Reference{
						APIVersion: "bindings.labs.vmware.com/v1alpha1",
						Kind:       "ProvisionedService",
						Name:       "my-service",
					},
					Env: []EnvVar{
						{Name: "MY_VAR", Key: "my-key1"},
						{Name: "MY_VAR", Key: "my-key2"},
					},
				},
			},
			expected: (&apis.FieldError{}).Also(
				apis.ErrMultipleOneOf(
					"spec.env[0].name",
					"spec.env[1].name",
				),
			),
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

func TestServiceBindingStatus_PropagateServiceBindingProjectionStatus(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name       string
		seed       *ServiceBindingStatus
		projection *labsinternalv1alpha1.ServiceBindingProjection
		expected   *ServiceBindingStatus
	}{
		{
			name:       "empty",
			seed:       &ServiceBindingStatus{},
			projection: nil,
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{Type: ServiceBindingConditionReady},
					{Type: ServiceBindingConditionServiceAvailable},
					{Type: ServiceBindingConditionProjectionReady},
				},
			},
		},
		{
			name:       "default",
			seed:       &ServiceBindingStatus{},
			projection: &labsinternalv1alpha1.ServiceBindingProjection{},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:               ServiceBindingConditionReady,
						Status:             metav1.ConditionUnknown,
						Reason:             "ProjectionReadyUnknown",
						LastTransitionTime: now,
					},
					{Type: ServiceBindingConditionServiceAvailable},
					{
						Type:               ServiceBindingConditionProjectionReady,
						Status:             metav1.ConditionUnknown,
						Reason:             "Unknown",
						LastTransitionTime: now,
					},
				},
			},
		},
		{
			name: "ready",
			seed: &ServiceBindingStatus{},
			projection: &labsinternalv1alpha1.ServiceBindingProjection{
				Status: labsinternalv1alpha1.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:   labsinternalv1alpha1.ServiceBindingProjectionConditionReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:               ServiceBindingConditionReady,
						Status:             metav1.ConditionUnknown,
						Reason:             "Unknown",
						LastTransitionTime: now,
					},
					{
						Type: ServiceBindingConditionServiceAvailable,
					},
					{
						Type:               ServiceBindingConditionProjectionReady,
						Status:             metav1.ConditionTrue,
						Reason:             "Projected",
						LastTransitionTime: now,
					},
				},
			},
		},
		{
			name: "not ready",
			seed: &ServiceBindingStatus{},
			projection: &labsinternalv1alpha1.ServiceBindingProjection{
				Status: labsinternalv1alpha1.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type:    labsinternalv1alpha1.ServiceBindingProjectionConditionReady,
								Status:  corev1.ConditionFalse,
								Reason:  "TheReason",
								Message: "the message",
							},
						},
					},
				},
			},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:               ServiceBindingConditionReady,
						Status:             metav1.ConditionFalse,
						Reason:             "ProjectionReadyTheReason",
						Message:            "the message",
						LastTransitionTime: now,
					},
					{
						Type: ServiceBindingConditionServiceAvailable,
					},
					{
						Type:               ServiceBindingConditionProjectionReady,
						Status:             metav1.ConditionFalse,
						Reason:             "TheReason",
						Message:            "the message",
						LastTransitionTime: now,
					},
				},
			},
		},
		{
			name: "unknown",
			seed: &ServiceBindingStatus{},
			projection: &labsinternalv1alpha1.ServiceBindingProjection{
				Status: labsinternalv1alpha1.ServiceBindingProjectionStatus{
					Status: duckv1.Status{
						Conditions: duckv1.Conditions{
							{
								Type: labsinternalv1alpha1.ServiceBindingProjectionConditionReady,
							},
						},
					},
				},
			},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:               ServiceBindingConditionReady,
						Status:             metav1.ConditionUnknown,
						Reason:             "ProjectionReadyUnknown",
						LastTransitionTime: now,
					},
					{
						Type: ServiceBindingConditionServiceAvailable,
					},
					{
						Type:               ServiceBindingConditionProjectionReady,
						Status:             metav1.ConditionUnknown,
						Reason:             "Unknown",
						LastTransitionTime: now,
					},
				},
			},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			actual := c.seed.DeepCopy()
			actual.InitializeConditions()
			actual.PropagateServiceBindingProjectionStatus(c.projection, now)
			if diff := cmp.Diff(c.expected, actual); diff != "" {
				t.Errorf("%s: PropagateServiceBindingProjectionStatus() (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}

func TestServiceBindingStatus_MarkServiceAvailable(t *testing.T) {
	now := metav1.Now()
	expected := &ServiceBindingStatus{
		Conditions: []metav1.Condition{
			{
				Type:               ServiceBindingConditionReady,
				Status:             metav1.ConditionUnknown,
				Reason:             "Unknown",
				LastTransitionTime: now,
			},
			{
				Type:               ServiceBindingConditionServiceAvailable,
				Status:             metav1.ConditionTrue,
				Reason:             "Available",
				LastTransitionTime: now,
			},
			{Type: ServiceBindingConditionProjectionReady},
		},
	}
	actual := &ServiceBindingStatus{}
	actual.InitializeConditions()
	actual.MarkServiceAvailable(now)

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("MarkServiceAvailable() (-expected, +actual): %s", diff)
	}
}

func TestServiceBindingStatus_MarkServiceUnavailable(t *testing.T) {
	now := metav1.Now()
	expected := &ServiceBindingStatus{
		Conditions: []metav1.Condition{
			{
				Type:               ServiceBindingConditionReady,
				Status:             metav1.ConditionFalse,
				Reason:             "ServiceAvailableTheReason",
				Message:            "the message",
				LastTransitionTime: now,
			},
			{
				Type:               ServiceBindingConditionServiceAvailable,
				Status:             metav1.ConditionFalse,
				Reason:             "TheReason",
				Message:            "the message",
				LastTransitionTime: now,
			},
			{Type: ServiceBindingConditionProjectionReady},
		},
	}
	actual := &ServiceBindingStatus{}
	actual.InitializeConditions()
	actual.MarkServiceUnavailable("TheReason", "the message", now)

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("MarkServiceUnavailable() (-expected, +actual): %s", diff)
	}
}

func TestServiceBindingStatus_InitializeConditions(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ServiceBindingStatus
		expected *ServiceBindingStatus
	}{
		{
			name: "empty",
			seed: &ServiceBindingStatus{},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{Type: ServiceBindingConditionReady},
					{Type: ServiceBindingConditionServiceAvailable},
					{Type: ServiceBindingConditionProjectionReady},
				},
			},
		},
		{
			name: "preserve",
			seed: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:   ServiceBindingConditionProjectionReady,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   ServiceBindingConditionReady,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   ServiceBindingConditionServiceAvailable,
						Status: metav1.ConditionTrue,
					},
				},
			},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:   ServiceBindingConditionReady,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   ServiceBindingConditionServiceAvailable,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   ServiceBindingConditionProjectionReady,
						Status: metav1.ConditionTrue,
					},
				},
			},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			actual := c.seed.DeepCopy()
			actual.InitializeConditions()
			if diff := cmp.Diff(c.expected, actual, cmpopts.IgnoreTypes(metav1.Time{})); diff != "" {
				t.Errorf("%s: InitializeConditions() (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}

func TestServiceBindingStatus_aggregateReadyCondition(t *testing.T) {
	now := metav1.Now()
	tests := []struct {
		name     string
		seed     *ServiceBindingStatus
		expected *ServiceBindingStatus
	}{
		{
			name: "empty",
			seed: &ServiceBindingStatus{},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:               ServiceBindingConditionReady,
						Status:             "Unknown",
						Reason:             "Unknown",
						LastTransitionTime: now,
					},
					{Type: ServiceBindingConditionServiceAvailable},
					{Type: ServiceBindingConditionProjectionReady},
				},
			},
		},
		{
			name: "Ready True",
			seed: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:   ServiceBindingConditionServiceAvailable,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   ServiceBindingConditionProjectionReady,
						Status: metav1.ConditionTrue,
					},
				},
			},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:               ServiceBindingConditionReady,
						Status:             metav1.ConditionTrue,
						Reason:             "Ready",
						LastTransitionTime: now,
					},
					{
						Type:   ServiceBindingConditionServiceAvailable,
						Status: metav1.ConditionTrue,
					},
					{
						Type:   ServiceBindingConditionProjectionReady,
						Status: metav1.ConditionTrue,
					},
				},
			},
		},
		{
			name: "ServiceAvailable False",
			seed: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:    ServiceBindingConditionServiceAvailable,
						Status:  metav1.ConditionFalse,
						Reason:  "TheReason",
						Message: "the message",
					},
					{
						Type:   ServiceBindingConditionProjectionReady,
						Status: metav1.ConditionTrue,
					},
				},
			},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:               ServiceBindingConditionReady,
						Status:             metav1.ConditionFalse,
						Reason:             "ServiceAvailableTheReason",
						Message:            "the message",
						LastTransitionTime: now,
					},
					{
						Type:    ServiceBindingConditionServiceAvailable,
						Status:  metav1.ConditionFalse,
						Reason:  "TheReason",
						Message: "the message",
					},
					{
						Type:   ServiceBindingConditionProjectionReady,
						Status: metav1.ConditionTrue,
					},
				},
			},
		},
		{
			name: "ServiceAvailable Unknown",
			seed: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:    ServiceBindingConditionServiceAvailable,
						Status:  metav1.ConditionUnknown,
						Reason:  "TheReason",
						Message: "the message",
					},
					{
						Type:   ServiceBindingConditionProjectionReady,
						Status: metav1.ConditionTrue,
					},
				},
			},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:               ServiceBindingConditionReady,
						Status:             metav1.ConditionUnknown,
						Reason:             "ServiceAvailableTheReason",
						Message:            "the message",
						LastTransitionTime: now,
					},
					{
						Type:    ServiceBindingConditionServiceAvailable,
						Status:  metav1.ConditionUnknown,
						Reason:  "TheReason",
						Message: "the message",
					},
					{
						Type:   ServiceBindingConditionProjectionReady,
						Status: metav1.ConditionTrue,
					},
				},
			},
		},
		{
			name: "ProjectionReady False",
			seed: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:   ServiceBindingConditionServiceAvailable,
						Status: metav1.ConditionTrue,
					},
					{
						Type:    ServiceBindingConditionProjectionReady,
						Status:  metav1.ConditionFalse,
						Reason:  "TheReason",
						Message: "the message",
					},
				},
			},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:               ServiceBindingConditionReady,
						Status:             metav1.ConditionFalse,
						Reason:             "ProjectionReadyTheReason",
						Message:            "the message",
						LastTransitionTime: now,
					},
					{
						Type:   ServiceBindingConditionServiceAvailable,
						Status: metav1.ConditionTrue,
					},
					{
						Type:    ServiceBindingConditionProjectionReady,
						Status:  metav1.ConditionFalse,
						Reason:  "TheReason",
						Message: "the message",
					},
				},
			},
		},
		{
			name: "ProjectionReady Unknown",
			seed: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:   ServiceBindingConditionServiceAvailable,
						Status: metav1.ConditionTrue,
					},
					{
						Type:    ServiceBindingConditionProjectionReady,
						Status:  metav1.ConditionUnknown,
						Reason:  "TheReason",
						Message: "the message",
					},
				},
			},
			expected: &ServiceBindingStatus{
				Conditions: []metav1.Condition{
					{
						Type:               ServiceBindingConditionReady,
						Status:             metav1.ConditionUnknown,
						Reason:             "ProjectionReadyTheReason",
						Message:            "the message",
						LastTransitionTime: now,
					},
					{
						Type:   ServiceBindingConditionServiceAvailable,
						Status: metav1.ConditionTrue,
					},
					{
						Type:    ServiceBindingConditionProjectionReady,
						Status:  metav1.ConditionUnknown,
						Reason:  "TheReason",
						Message: "the message",
					},
				},
			},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			actual := c.seed.DeepCopy()
			actual.InitializeConditions()
			actual.aggregateReadyCondition(now)
			if diff := cmp.Diff(c.expected, actual); diff != "" {
				t.Errorf("%s: aggregateReadyCondition() (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}
