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
	"github.com/stretchr/testify/assert"
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
				apis.ErrMissingField("spec.application"),
				apis.ErrMissingField("spec.service"),
			),
		},
		{
			name: "valid",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Application: &ApplicationReference{
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
				},
			},
			expected: nil,
		},
		{
			name: "valid, application selector",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Application: &ApplicationReference{
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
					Application: &ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-application",
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
					Application: &ApplicationReference{
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
					Application: &ApplicationReference{
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
				apis.ErrDisallowedFields("spec.application.namespace"),
				apis.ErrDisallowedFields("spec.service.namespace"),
			),
		},
		{
			name: "empty env",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Application: &ApplicationReference{
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
					Application: &ApplicationReference{
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
		{
			name: "disallow status annotations",
			seed: &ServiceBinding{
				Spec: ServiceBindingSpec{
					Application: &ApplicationReference{
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
				},
				Status: ServiceBindingStatus{
					Status: duckv1.Status{
						Annotations: map[string]string{},
					},
				},
			},
			expected: (&apis.FieldError{}).Also(
				apis.ErrDisallowedFields("status.annotations"),
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

func TestServiceBinding_GetStatus(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ServiceBinding
		expected *duckv1.Status
	}{
		{
			name:     "empty",
			seed:     &ServiceBinding{},
			expected: &duckv1.Status{},
		},
		{
			name: "status",
			seed: &ServiceBinding{
				Status: ServiceBindingStatus{
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

func TestServiceBinding_GetConditionSet(t *testing.T) {
	expected := sbCondSet
	actual := (&ServiceBinding{}).GetConditionSet()
	assert.Exactly(t, expected, actual)
}

func TestServiceBindingStatus_PropagateServiceBindingProjectionStatus(t *testing.T) {
	tests := []struct {
		name       string
		seed       *ServiceBindingStatus
		projection *labsinternalv1alpha1.ServiceBindingProjection
		expected   *duckv1.Status
	}{
		{
			name:       "empty",
			seed:       &ServiceBindingStatus{},
			projection: nil,
			expected:   &duckv1.Status{},
		},
		{
			name:       "default",
			seed:       &ServiceBindingStatus{},
			projection: &labsinternalv1alpha1.ServiceBindingProjection{},
			expected:   &duckv1.Status{},
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
			expected: &duckv1.Status{
				Conditions: duckv1.Conditions{
					{
						Type:   ServiceBindingConditionProjectionReady,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   ServiceBindingConditionReady,
						Status: corev1.ConditionUnknown,
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
								Message: "TheMessage",
								Reason:  "a reason",
							},
						},
					},
				},
			},
			expected: &duckv1.Status{
				Conditions: duckv1.Conditions{
					{
						Type:    ServiceBindingConditionProjectionReady,
						Status:  corev1.ConditionFalse,
						Message: "TheMessage",
						Reason:  "a reason",
					},
					{
						Type:    ServiceBindingConditionReady,
						Status:  corev1.ConditionFalse,
						Message: "TheMessage",
						Reason:  "a reason",
					},
				},
			},
		},
		{
			name: "unkown",
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
			expected: &duckv1.Status{
				Conditions: duckv1.Conditions{
					{
						Type:   ServiceBindingConditionProjectionReady,
						Status: corev1.ConditionUnknown,
					},
					{
						Type:   ServiceBindingConditionReady,
						Status: corev1.ConditionUnknown,
					},
				},
			},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			actual := c.seed.DeepCopy()
			actual.PropagateServiceBindingProjectionStatus(c.projection)
			if diff := cmp.Diff(c.expected, &actual.Status, cmpopts.IgnoreTypes(apis.VolatileTime{})); diff != "" {
				t.Errorf("%s: PropagateServiceBindingProjectionStatus() (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}

func TestServiceBindingStatus_MarkServiceAvailable(t *testing.T) {
	expected := &ServiceBindingStatus{
		Status: duckv1.Status{
			Conditions: duckv1.Conditions{
				{
					Type:   ServiceBindingConditionReady,
					Status: corev1.ConditionUnknown,
				},
				{
					Type:   ServiceBindingConditionServiceAvailable,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	actual := &ServiceBindingStatus{}
	actual.MarkServiceAvailable()

	if diff := cmp.Diff(expected, actual, cmpopts.IgnoreTypes(apis.VolatileTime{})); diff != "" {
		t.Errorf("MarkServiceAvailable() (-expected, +actual): %s", diff)
	}
}

func TestServiceBindingStatus_MarkServiceUnavailable(t *testing.T) {
	expected := &ServiceBindingStatus{
		Status: duckv1.Status{
			Conditions: duckv1.Conditions{
				{
					Type:    ServiceBindingConditionReady,
					Status:  corev1.ConditionFalse,
					Reason:  "TheReason",
					Message: "a message",
				},
				{
					Type:    ServiceBindingConditionServiceAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "TheReason",
					Message: "a message",
				},
			},
		},
	}
	actual := &ServiceBindingStatus{}
	actual.MarkServiceUnavailable("TheReason", "a message")

	if diff := cmp.Diff(expected, actual, cmpopts.IgnoreTypes(apis.VolatileTime{})); diff != "" {
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
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   ServiceBindingConditionProjectionReady,
							Status: corev1.ConditionUnknown,
						},
						{
							Type:   ServiceBindingConditionReady,
							Status: corev1.ConditionUnknown,
						},
						{
							Type:   ServiceBindingConditionServiceAvailable,
							Status: corev1.ConditionUnknown,
						},
					},
				},
			},
		},
		{
			name: "preserve",
			seed: &ServiceBindingStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   ServiceBindingConditionReady,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   ServiceBindingConditionProjectionReady,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   ServiceBindingConditionServiceAvailable,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: &ServiceBindingStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   ServiceBindingConditionReady,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   ServiceBindingConditionProjectionReady,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   ServiceBindingConditionServiceAvailable,
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
