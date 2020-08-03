/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha2

import (
	"context"
	"crypto/sha1"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/tracker"
)

const (
	ServiceBindingProjectionConditionReady                = apis.ConditionReady
	ServiceBindingProjectionConditionApplicationAvailable = "ApplicationAvailable"
)

var sbpCondSet = apis.NewLivingConditionSet(
	ServiceBindingProjectionConditionApplicationAvailable,
)

func (b *ServiceBindingProjection) GetStatus() *duckv1.Status {
	return &b.Status.Status
}

func (b *ServiceBindingProjection) GetConditionSet() apis.ConditionSet {
	return sbpCondSet
}

func (b *ServiceBindingProjection) GetSubject() tracker.Reference {
	var ref tracker.Reference
	b.Spec.Application.Reference.DeepCopyInto(&ref)
	ref.Namespace = b.Namespace
	return ref
}

func (b *ServiceBindingProjection) GetBindingStatus() duck.BindableStatus {
	return &b.Status
}

const (
	ServiceBindingProjectionTypeKey    = "projection.service.binding/type"
	ServiceBindingProjectionTypeCustom = "Custom"
)

func (b *ServiceBindingProjection) IsCustomProjection() bool {
	return b.Annotations != nil && b.Annotations[ServiceBindingProjectionTypeKey] == ServiceBindingProjectionTypeCustom
}

func (b *ServiceBindingProjection) Do(ctx context.Context, ps *duckv1.WithPod) {
	// undo existing bindings so we can start clean
	b.Undo(ctx, ps)

	if b.IsCustomProjection() {
		// someone else is responsible for the projection
		return
	}

	existingVolumes := sets.NewString()
	for _, v := range ps.Spec.Template.Spec.Volumes {
		existingVolumes.Insert(v.Name)
	}

	newVolumes := sets.NewString()
	sb := b.Spec.Binding

	bindingVolume := truncateAt63("binding-%x", sha1.Sum([]byte(sb.Name)))
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
			b.doContainer(ctx, ps, c, bindingVolume, sb.Name)
		}
	}
	for i := range ps.Spec.Template.Spec.Containers {
		c := &ps.Spec.Template.Spec.Containers[i]
		if b.isTargetContainer(i, c) {
			b.doContainer(ctx, ps, c, bindingVolume, sb.Name)
		}
	}

	// track which volumes are injected, so they can be removed when no longer used
	ps.Annotations[b.annotationKey()] = strings.Join(newVolumes.List(), ",")
}

func (b *ServiceBindingProjection) doContainer(ctx context.Context, ps *duckv1.WithPod, c *corev1.Container, bindingVolume, secretName string) {
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

func (b *ServiceBindingProjection) Undo(ctx context.Context, ps *duckv1.WithPod) {
	if ps.Annotations == nil {
		ps.Annotations = map[string]string{}
	}

	key := b.annotationKey()

	boundVolumes := sets.NewString(strings.Split(ps.Annotations[key], ",")...)
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
		b.undoContainer(ctx, ps, &ps.Spec.Template.Spec.InitContainers[i], boundVolumes, boundSecrets)
	}
	for i := range ps.Spec.Template.Spec.Containers {
		b.undoContainer(ctx, ps, &ps.Spec.Template.Spec.Containers[i], boundVolumes, boundSecrets)
	}

	delete(ps.Annotations, key)
}

func (b *ServiceBindingProjection) undoContainer(ctx context.Context, ps *duckv1.WithPod, c *corev1.Container, boundVolumes, boundSecrets sets.String) {
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
	return truncateAt63("%s-%x", ServiceBindingProjectionAnnotationKey, sha1.Sum([]byte(b.Name)))
}

func (bs *ServiceBindingProjectionStatus) InitializeConditions() {
	sbpCondSet.Manage(bs).InitializeConditions()
}

func (bs *ServiceBindingProjectionStatus) MarkBindingAvailable() {
	sbpCondSet.Manage(bs).MarkTrue(ServiceBindingProjectionConditionApplicationAvailable)
}

func (bs *ServiceBindingProjectionStatus) MarkBindingUnavailable(reason string, message string) {
	if strings.HasPrefix(reason, "Subject") {
		// knative/pkg uses "Subject*" reasons, we want to rename to "Application*"
		reason = strings.Replace(reason, "Subject", "Application", 1)
	}
	sbpCondSet.Manage(bs).MarkFalse(
		ServiceBindingProjectionConditionApplicationAvailable, reason, message)
}

func (bs *ServiceBindingProjectionStatus) SetObservedGeneration(gen int64) {
	bs.ObservedGeneration = gen
}

func truncateAt63(msg string, a ...interface{}) string {
	s := fmt.Sprintf(msg, a...)
	if len(s) <= 63 {
		return s
	}
	return s[:63]
}
