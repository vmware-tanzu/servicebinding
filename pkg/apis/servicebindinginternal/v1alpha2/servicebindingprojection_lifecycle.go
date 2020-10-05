/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha2

import (
	"context"
	"crypto/sha1"
	"fmt"
	"sort"
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

	ServiceBindingRootEnv = "SERVICE_BINDING_ROOT"
	bindingVolumePrefix   = "binding-"
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

	injectedSecrets, injectedVolumes := b.injectedValues(ps)

	sb := b.Spec.Binding

	volume := corev1.Volume{
		Name: truncateAt63("%s%x", bindingVolumePrefix, sha1.Sum([]byte(sb.Name))),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: sb.Name,
			},
		},
	}
	ps.Spec.Template.Spec.Volumes = append(ps.Spec.Template.Spec.Volumes, volume)
	injectedSecrets.Insert(volume.Secret.SecretName)
	injectedVolumes.Insert(volume.Name)
	sort.SliceStable(ps.Spec.Template.Spec.Volumes, func(i, j int) bool {
		iname := ps.Spec.Template.Spec.Volumes[i].Name
		jname := ps.Spec.Template.Spec.Volumes[j].Name
		// only sort injected volumes
		if !injectedVolumes.HasAll(iname, jname) {
			return false
		}
		return iname < jname
	})
	// track which secret is injected, so it can be removed when no longer used
	ps.Annotations[b.annotationKey()] = volume.Secret.SecretName

	for i := range ps.Spec.Template.Spec.InitContainers {
		c := &ps.Spec.Template.Spec.InitContainers[i]
		if b.isTargetContainer(-1, c) {
			b.doContainer(ctx, ps, c, volume.Name, sb.Name, injectedVolumes, injectedSecrets)
		}
	}
	for i := range ps.Spec.Template.Spec.Containers {
		c := &ps.Spec.Template.Spec.Containers[i]
		if b.isTargetContainer(i, c) {
			b.doContainer(ctx, ps, c, volume.Name, sb.Name, injectedVolumes, injectedSecrets)
		}
	}
}

func (b *ServiceBindingProjection) doContainer(ctx context.Context, ps *duckv1.WithPod, c *corev1.Container, bindingVolume, secretName string, allInjectedVolumes, allInjectedSecrets sets.String) {
	mountPath := ""
	// lookup predefined mount path
	for _, e := range c.Env {
		if e.Name == ServiceBindingRootEnv {
			mountPath = e.Value
			break
		}
	}
	if mountPath == "" {
		// default mount path
		mountPath = "/bindings"
		c.Env = append(c.Env, corev1.EnvVar{
			Name:  ServiceBindingRootEnv,
			Value: mountPath,
		})
	}

	// inject metadata
	c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
		Name:      bindingVolume,
		MountPath: fmt.Sprintf("%s/%s", mountPath, b.Spec.Name),
		ReadOnly:  true,
	})
	sort.SliceStable(c.VolumeMounts, func(i, j int) bool {
		iname := c.VolumeMounts[i].Name
		jname := c.VolumeMounts[j].Name
		// only sort injected volume mounts
		if !allInjectedVolumes.HasAll(iname, jname) {
			return false
		}
		return iname < jname
	})

	if len(b.Spec.Env) != 0 {
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
		sort.SliceStable(c.Env, func(i, j int) bool {
			iv := c.Env[i]
			jv := c.Env[j]
			// only sort injected env
			if iv.ValueFrom == nil || iv.ValueFrom.SecretKeyRef == nil ||
				jv.ValueFrom == nil || jv.ValueFrom.SecretKeyRef == nil ||
				!allInjectedSecrets.HasAll(iv.ValueFrom.SecretKeyRef.Name, jv.ValueFrom.SecretKeyRef.Name) {
				return false
			}
			return iv.Name < jv.Name
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
	removeSecret := ps.Annotations[key]
	removeVolume := ""
	delete(ps.Annotations, key)

	preservedVolumes := []corev1.Volume{}
	for _, v := range ps.Spec.Template.Spec.Volumes {
		if v.Secret != nil && v.Secret.SecretName == removeSecret {
			removeVolume = v.Name
			continue
		}
		preservedVolumes = append(preservedVolumes, v)
	}
	ps.Spec.Template.Spec.Volumes = preservedVolumes

	for i := range ps.Spec.Template.Spec.InitContainers {
		b.undoContainer(ctx, ps, &ps.Spec.Template.Spec.InitContainers[i], removeSecret, removeVolume)
	}
	for i := range ps.Spec.Template.Spec.Containers {
		b.undoContainer(ctx, ps, &ps.Spec.Template.Spec.Containers[i], removeSecret, removeVolume)
	}
}

func (b *ServiceBindingProjection) undoContainer(ctx context.Context, ps *duckv1.WithPod, c *corev1.Container, removeSecret, removeVolume string) {
	preservedMounts := []corev1.VolumeMount{}
	for _, vm := range c.VolumeMounts {
		if removeVolume != vm.Name {
			preservedMounts = append(preservedMounts, vm)
		}
	}
	c.VolumeMounts = preservedMounts

	preservedEnv := []corev1.EnvVar{}
	for _, e := range c.Env {
		if e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil || e.ValueFrom.SecretKeyRef.Name != removeSecret {
			preservedEnv = append(preservedEnv, e)
		}
	}
	c.Env = preservedEnv
}

func (b *ServiceBindingProjection) annotationKey() string {
	return truncateAt63("%s-%x", ServiceBindingProjectionAnnotationKey, sha1.Sum([]byte(b.Name)))
}

func (b *ServiceBindingProjection) injectedValues(ps *duckv1.WithPod) (sets.String, sets.String) {
	secrets := sets.NewString()
	volumes := sets.NewString()
	for k, v := range ps.Annotations {
		if strings.HasPrefix(k, ServiceBindingProjectionAnnotationKey) {
			secrets.Insert(v)
		}
	}
	for _, v := range ps.Spec.Template.Spec.Volumes {
		if v.Secret != nil && secrets.Has(v.Secret.SecretName) {
			volumes.Insert(v.Name)
		}
	}
	return secrets, volumes
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
