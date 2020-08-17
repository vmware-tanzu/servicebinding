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
	"k8s.io/apimachinery/pkg/util/intstr"
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

func TestServiceBindingProjection_Undo(t *testing.T) {
	tests := []struct {
		name     string
		binding  *ServiceBindingProjection
		seed     *duckv1.WithPod
		expected *duckv1.WithPod
	}{
		{
			name:     "empty",
			binding:  &ServiceBindingProjection{},
			seed:     &duckv1.WithPod{},
			expected: &duckv1.WithPod{},
		},
		{
			name: "remove bound volumes",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
			},
			seed: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "injected-a,injected-b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{Name: "preserve"},
								{Name: "injected-a"},
								{Name: "injected-b"},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{Name: "preserve"},
							},
						},
					},
				},
			},
		},
		{
			name: "remove injected container volumemounts",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
			},
			seed: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "injected",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									VolumeMounts: []corev1.VolumeMount{
										{Name: "preserve"},
										{Name: "injected"},
									},
								},
							},
							Containers: []corev1.Container{
								{
									VolumeMounts: []corev1.VolumeMount{
										{Name: "preserve"},
										{Name: "injected"},
									},
								},
							},
							Volumes: []corev1.Volume{
								{Name: "injected"},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									VolumeMounts: []corev1.VolumeMount{
										{Name: "preserve"},
									},
								},
							},
							Containers: []corev1.Container{
								{
									VolumeMounts: []corev1.VolumeMount{
										{Name: "preserve"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "remove injected environment variables",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
			},
			seed: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "injected",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name: "PRESERVE",
										},
										{
											Name: "INJECTED",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: "injected-secret",
													},
												},
											},
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name: "PRESERVE",
										},
										{
											Name: "INJECTED",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: "injected-secret",
													},
												},
											},
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "injected",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "injected-secret",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{Name: "PRESERVE"},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{Name: "PRESERVE"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "undoes previous bindings even if a custom projections",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
					Annotations: map[string]string{
						"projection.service.binding/type": "Custom",
					},
				},
			},
			seed: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "injected-a,injected-b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{Name: "preserve"},
								{Name: "injected-a"},
								{Name: "injected-b"},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{Name: "preserve"},
							},
						},
					},
				},
			},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			actual := c.seed.DeepCopy()
			binding := c.binding.DeepCopy()
			binding.Undo(context.TODO(), actual)
			if diff := cmp.Diff(c.binding, binding); diff != "" {
				t.Errorf("%s: Undo() unexpected binding mutation (-expected, +actual): %s", c.name, diff)
			}
			if diff := cmp.Diff(c.expected, actual, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("%s: Undo() (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}

func TestServiceBindingProjection_Do(t *testing.T) {
	tests := []struct {
		name     string
		binding  *ServiceBindingProjection
		seed     *duckv1.WithPod
		expected *duckv1.WithPod
	}{
		{
			name: "inject volume into each container",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding-name",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
			seed: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{},
							},
							Containers: []corev1.Container{
								{},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "my-secret",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "inject volume into named container",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding-name",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Application: ApplicationReference{
						Containers: []intstr.IntOrString{
							intstr.FromString("my-container"),
						},
					},
				},
			},
			seed: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{},
								{Name: "my-container"},
							},
							Containers: []corev1.Container{
								{},
								{Name: "my-container"},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{},
								{
									Name: "my-container",
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Containers: []corev1.Container{
								{},
								{
									Name: "my-container",
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "my-secret",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "inject volume into a container by index",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding-name",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Application: ApplicationReference{
						Containers: []intstr.IntOrString{
							intstr.FromInt(1),
						},
					},
				},
			},
			seed: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{},
								{},
							},
							Containers: []corev1.Container{
								{},
								{},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{},
								{},
							},
							Containers: []corev1.Container{
								{},
								{
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "my-secret",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "preserve volume mounts",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding-name",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
			seed: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									VolumeMounts: []corev1.VolumeMount{
										{Name: "preserve"},
									},
								},
							},
							Volumes: []corev1.Volume{
								{Name: "preserve"},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{Name: "preserve"},
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{Name: "preserve"},
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "my-secret",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "inject volume at custom path",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding-name",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
			seed: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/custom/path",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/custom/path",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/custom/path/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "my-secret",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "inject custom envvars",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding-name",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
					Env: []EnvVar{
						{
							Name: "MY_VAR",
							Key:  "my-key",
						},
					},
				},
			},
			seed: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name: "PRESERVE",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name: "PRESERVE",
										},
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
										{
											Name: "MY_VAR",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{
														Name: "my-secret",
													},
													Key: "my-key",
												},
											},
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "my-secret",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "runs undo before do",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding-name",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
			seed: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "injected",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{Name: "preserve"},
								{Name: "injected"},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								{Name: "preserve"},
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "my-secret",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "idempotent",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
				},
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding-name",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
			seed: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "my-secret",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"internal.service.binding/projection-16384e6a11df69776193b6a877b": "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
					},
				},
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "SERVICE_BINDING_ROOT",
											Value: "/bindings",
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
											MountPath: "/bindings/my-binding-name",
											ReadOnly:  true,
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "binding-5c5a15a8b0b3e154d77746945e563ba40100681b",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: "my-secret",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "don't bind custom projections",
			binding: &ServiceBindingProjection{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-binding",
					Annotations: map[string]string{
						"projection.service.binding/type": "Custom",
					},
				},
				Spec: ServiceBindingProjectionSpec{
					Name: "my-binding-name",
					Binding: corev1.LocalObjectReference{
						Name: "my-secret",
					},
				},
			},
			seed: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{},
							},
							Containers: []corev1.Container{
								{},
							},
						},
					},
				},
			},
			expected: &duckv1.WithPod{
				Spec: duckv1.WithPodSpec{
					Template: duckv1.PodSpecable{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{},
							},
							Containers: []corev1.Container{
								{},
							},
						},
					},
				},
			},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			actual := c.seed.DeepCopy()
			binding := c.binding.DeepCopy()
			binding.Do(context.TODO(), actual)
			if diff := cmp.Diff(c.binding, binding); diff != "" {
				t.Errorf("%s: Do() unexpected binding mutation (-expected, +actual): %s", c.name, diff)
			}
			if diff := cmp.Diff(c.expected, actual, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("%s: Do() (-expected, +actual): %s", c.name, diff)
			}
		})
	}
}
