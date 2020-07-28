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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/tracker"
)

func TestServiceBindingProjection_GetGroupVersionKind(t *testing.T) {
	if got, want := (&ServiceBindingProjection{}).GetGroupVersionKind().String(), "internal.service.binding/v1alpha2, Kind=ServiceBindingProjection"; got != want {
		t.Errorf("GetGroupVersionKind() = %v, want %v", got, want)
	}
}

func TestServiceBindingProjection_SetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ServiceBindingProjection
		expected *ServiceBindingProjection
	}{
		{
			name: "empty",
			seed: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
			},
			expected: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
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

func TestServiceBindingProjection_Validate(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ServiceBindingProjection
		expected *apis.FieldError
	}{
		{
			name: "empty",
			seed: &ServiceBindingProjection{},
			expected: (&apis.FieldError{}).Also(
				apis.ErrMissingOneOf(
					"spec.application.name",
					"spec.application.selector",
				),
				apis.ErrMissingField(
					"spec.application.apiVersion",
					"spec.application.kind",
				),
				apis.ErrMissingField("spec.binding"),
				apis.ErrMissingField("spec.name"),
			),
		},
		{
			name: "valid",
			seed: &ServiceBindingProjection{
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Application: ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid, application selector",
			seed: &ServiceBindingProjection{
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Application: ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Selector:   &metav1.LabelSelector{},
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid, env",
			seed: &ServiceBindingProjection{
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Application: ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
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
			seed: &ServiceBindingProjection{
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Application: ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
							Namespace:  "default",
						},
					},
				},
			},
			expected: (&apis.FieldError{}).Also(
				apis.ErrDisallowedFields("spec.application.namespace"),
			),
		},
		{
			name: "empty env",
			seed: &ServiceBindingProjection{
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Application: ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
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
			seed: &ServiceBindingProjection{
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Application: ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
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
			seed: &ServiceBindingProjection{
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Application: ApplicationReference{
						Reference: tracker.Reference{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       "my-app",
						},
					},
				},
				Status: ServiceBindingProjectionStatus{
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

func TestServiceBindingProjection_GetStatus(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ServiceBindingProjection
		expected *duckv1.Status
	}{
		{
			name:     "empty",
			seed:     &ServiceBindingProjection{},
			expected: &duckv1.Status{},
		},
		{
			name: "status",
			seed: &ServiceBindingProjection{
				Status: ServiceBindingProjectionStatus{
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

func TestServiceBindingProjection_GetConditionSet(t *testing.T) {
	expected := sbpCondSet
	actual := (&ServiceBindingProjection{}).GetConditionSet()
	assert.Exactly(t, expected, actual)
}

func TestServiceBindingProjectionStatus_MarBindingAvailable(t *testing.T) {
	expected := &ServiceBindingProjectionStatus{
		Status: duckv1.Status{
			Conditions: duckv1.Conditions{
				{
					Type:   ServiceBindingProjectionConditionApplicationAvailable,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   ServiceBindingProjectionConditionReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	actual := &ServiceBindingProjectionStatus{}
	actual.MarkBindingAvailable()

	if diff := cmp.Diff(expected, actual, cmpopts.IgnoreTypes(apis.VolatileTime{})); diff != "" {
		t.Errorf("MarkServiceAvailable() (-expected, +actual): %s", diff)
	}
}

func TestServiceBindingProjectionStatus_MarkBindingUnavailable(t *testing.T) {
	expected := &ServiceBindingProjectionStatus{
		Status: duckv1.Status{
			Conditions: duckv1.Conditions{
				{
					Type:    ServiceBindingProjectionConditionApplicationAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "ApplicationReason",
					Message: "a message",
				},
				{
					Type:    ServiceBindingProjectionConditionReady,
					Status:  corev1.ConditionFalse,
					Reason:  "ApplicationReason",
					Message: "a message",
				},
			},
		},
	}
	actual := &ServiceBindingProjectionStatus{}
	actual.MarkBindingUnavailable("SubjectReason", "a message")

	if diff := cmp.Diff(expected, actual, cmpopts.IgnoreTypes(apis.VolatileTime{})); diff != "" {
		t.Errorf("MarkBindingUnavailable() (-expected, +actual): %s", diff)
	}
}

func TestServiceBindingProjectionStatus_InitializeConditions(t *testing.T) {
	tests := []struct {
		name     string
		seed     *ServiceBindingProjectionStatus
		expected *ServiceBindingProjectionStatus
	}{
		{
			name: "empty",
			seed: &ServiceBindingProjectionStatus{},
			expected: &ServiceBindingProjectionStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   ServiceBindingProjectionConditionApplicationAvailable,
							Status: corev1.ConditionUnknown,
						},
						{
							Type:   ServiceBindingProjectionConditionReady,
							Status: corev1.ConditionUnknown,
						},
					},
				},
			},
		},
		{
			name: "preserve",
			seed: &ServiceBindingProjectionStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   ServiceBindingProjectionConditionReady,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   ServiceBindingProjectionConditionApplicationAvailable,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: &ServiceBindingProjectionStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Type:   ServiceBindingProjectionConditionReady,
							Status: corev1.ConditionTrue,
						},
						{
							Type:   ServiceBindingProjectionConditionApplicationAvailable,
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

func TestServiceBindingProjection_GetSubject(t *testing.T) {
	expected := tracker.Reference{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Namespace:  "default",
		Name:       "my-application",
	}
	seed := &ServiceBindingProjection{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
		},
		Spec: ServiceBindingProjectionSpec{
			Application: ApplicationReference{
				Reference: tracker.Reference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "my-application",
				},
			},
		},
	}
	actual := seed.GetSubject()

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("GetSubject() (-expected, +actual): %s", diff)
	}
}

func TestServiceBindingProjection_GetBindingStatus(t *testing.T) {
	seed := &ServiceBindingProjection{
		Status: ServiceBindingProjectionStatus{
			Status: duckv1.Status{
				ObservedGeneration: 1,
			},
		},
	}
	expected := &seed.Status
	actual := seed.GetBindingStatus()
	assert.Same(t, expected, actual)
}

func TestServiceBindingProjection_SetObservedGeneration(t *testing.T) {
	seed := &ServiceBindingProjection{}
	expected := int64(1)
	seed.Status.SetObservedGeneration(expected)
	actual := seed.Status.ObservedGeneration
	assert.Equal(t, expected, actual)
}
