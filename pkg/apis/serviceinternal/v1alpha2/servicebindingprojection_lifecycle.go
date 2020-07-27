/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha2

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	v1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/tracker"
)

const (
	ServiceBindingConditionReady                = apis.ConditionReady
	ServiceBindingConditionApplicationAvailable = "ApplicationAvailable"
)

var serviceCondSet = apis.NewLivingConditionSet(
	ServiceBindingConditionApplicationAvailable,
)

func (b *ServiceBindingProjection) GetStatus() *duckv1.Status {
	return &b.Status.Status
}

func (b *ServiceBindingProjection) GetConditionSet() apis.ConditionSet {
	return serviceCondSet
}

func (b *ServiceBindingProjection) GetSubject() tracker.Reference {
	return b.Spec.Application.Reference
}

func (b *ServiceBindingProjection) GetBindingStatus() duck.BindableStatus {
	return &b.Status
}

func (b *ServiceBindingProjection) Do(ctx context.Context, ps *v1.WithPod) {
	// undo existing bindings so we can start clean
	b.Undo(ctx, ps)

	existingVolumes := sets.NewString()
	for _, v := range ps.Spec.Template.Spec.Volumes {
		existingVolumes.Insert(v.Name)
	}

	newVolumes := sets.NewString()
	sb := b.Spec.Binding

	// TODO ensure unique volume names
	// TODO limit volume name length
	bindingVolume := fmt.Sprintf("%s-binding", sb.Name)
	if !existingVolumes.Has(bindingVolume) {
		ps.Spec.Template.Spec.Volumes = append(ps.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: bindingVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: sb.Name,
				},
			},
		})
		existingVolumes.Insert(bindingVolume)
		newVolumes.Insert(bindingVolume)
	}
	for i := range ps.Spec.Template.Spec.InitContainers {
		c := &ps.Spec.Template.Spec.InitContainers[i]
		if b.isTargetContainer(-1, c) {
			b.DoContainer(ctx, ps, c, bindingVolume, sb.Name)
		}
	}
	for i := range ps.Spec.Template.Spec.Containers {
		c := &ps.Spec.Template.Spec.Containers[i]
		if b.isTargetContainer(i, c) {
			b.DoContainer(ctx, ps, c, bindingVolume, sb.Name)
		}
	}

	// track which volumes are injected, so they can be removed when no longer used
	ps.Annotations[b.annotationKey()] = strings.Join(newVolumes.List(), ",")
}

func (b *ServiceBindingProjection) DoContainer(ctx context.Context, ps *v1.WithPod, c *corev1.Container, bindingVolume, secretName string) {
	mountPath := ""
	// lookup predefined mount path
	for _, e := range c.Env {
		if e.Name == "SERVICE_BINDINGS_ROOT" {
			mountPath = e.Value
			break
		}
	}
	if mountPath == "" {
		// default mount path
		mountPath = "/bindings"
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  "SERVICE_BINDINGS_ROOT",
			Value: mountPath,
		})
	}

	containerVolumes := sets.NewString()
	for _, vm := range c.VolumeMounts {
		containerVolumes.Insert(vm.Name)
	}

	if !containerVolumes.Has(bindingVolume) {
		// inject metadata
		c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
			Name:      bindingVolume,
			MountPath: fmt.Sprintf("%s/%s", mountPath, b.Spec.Name),
			ReadOnly:  true,
		})
	}

	for _, e := range b.Spec.Env {
		c.Env = append(c.Env, corev1.EnvVar{
			Name: e.Name,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: e.Key,
				},
			},
		})
	}
}

func (b *ServiceBindingProjection) isTargetContainer(idx int, c *corev1.Container) bool {
	targets := b.Spec.Application.Containers
	if len(targets) == 0 {
		return true
	}
	for _, t := range targets {
		switch t.Type {
		case intstr.Int:
			if idx == t.IntValue() {
				return true
			}
		case intstr.String:
			if c.Name == t.String() {
				return true
			}
		}
	}
	return false
}

func (b *ServiceBindingProjection) Undo(ctx context.Context, ps *v1.WithPod) {
	if ps.Annotations == nil {
		ps.Annotations = map[string]string{}
	}

	boundVolumes := sets.NewString(strings.Split(ps.Annotations[b.annotationKey()], ",")...)
	boundSecrets := sets.NewString()

	preservedVolumes := []corev1.Volume{}
	for _, v := range ps.Spec.Template.Spec.Volumes {
		if !boundVolumes.Has(v.Name) {
			preservedVolumes = append(preservedVolumes, v)
			continue
		}
		if v.Secret != nil {
			boundSecrets = boundSecrets.Insert(v.Secret.SecretName)
		}
	}
	ps.Spec.Template.Spec.Volumes = preservedVolumes

	for i := range ps.Spec.Template.Spec.InitContainers {
		b.UndoContainer(ctx, ps, &ps.Spec.Template.Spec.InitContainers[i], boundVolumes, boundSecrets)
	}
	for i := range ps.Spec.Template.Spec.Containers {
		b.UndoContainer(ctx, ps, &ps.Spec.Template.Spec.Containers[i], boundVolumes, boundSecrets)
	}

	delete(ps.Annotations, b.annotationKey())
}

func (b *ServiceBindingProjection) UndoContainer(ctx context.Context, ps *v1.WithPod, c *corev1.Container, boundVolumes, boundSecrets sets.String) {
	preservedMounts := []corev1.VolumeMount{}
	for _, vm := range c.VolumeMounts {
		if !boundVolumes.Has(vm.Name) {
			preservedMounts = append(preservedMounts, vm)
		}
	}
	c.VolumeMounts = preservedMounts

	preservedEnv := []corev1.EnvVar{}
	for _, e := range c.Env {
		if e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil || !boundSecrets.Has(e.ValueFrom.SecretKeyRef.Name) {
			preservedEnv = append(preservedEnv, e)
		}
	}
	c.Env = preservedEnv
}

func (b *ServiceBindingProjection) annotationKey() string {
	return fmt.Sprintf("%s-%s", ServiceBindingProjectionAnnotationKey, b.Name)
}

func (bs *ServiceBindingProjectionStatus) InitializeConditions() {
	serviceCondSet.Manage(bs).InitializeConditions()
}

func (bs *ServiceBindingProjectionStatus) MarkBindingAvailable() {
	serviceCondSet.Manage(bs).MarkTrue(ServiceBindingConditionApplicationAvailable)
}

func (bs *ServiceBindingProjectionStatus) MarkBindingUnavailable(reason string, message string) {
	if strings.HasPrefix(reason, "Subject") {
		// knative/pkg uses "Subject*" reasons, we want to rename to "Application*"
		reason = strings.Replace(reason, "Subject", "Application", 1)
	}
	serviceCondSet.Manage(bs).MarkFalse(
		ServiceBindingConditionApplicationAvailable, reason, message)
}

func (bs *ServiceBindingProjectionStatus) SetObservedGeneration(gen int64) {
	bs.ObservedGeneration = gen
}
