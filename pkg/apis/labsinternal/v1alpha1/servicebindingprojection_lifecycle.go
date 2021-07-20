/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
	"crypto/sha1"
	"fmt"
	"regexp"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
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

func (b *ServiceBindingProjection) Do(ctx context.Context, ps *duckv1.WithPod) {
	// undo existing bindings so we can start clean
	b.Undo(ctx, ps)

	injectedSecrets, injectedVolumes := b.injectedValues(ps)
	key := b.annotationKey()

	sb := b.Spec.Binding

	volume := corev1.Volume{
		Name: fmt.Sprintf("%s%x", bindingVolumePrefix, sha1.Sum([]byte(sb.Name))),
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: sb.Name,
							},
						},
					},
				},
			},
		},
	}
	if b.Spec.Type != "" {
		typeAnnotation := fmt.Sprintf("%s-type", key)
		ps.Spec.Template.Annotations[typeAnnotation] = b.Spec.Type
		volume.VolumeSource.Projected.Sources = append(volume.VolumeSource.Projected.Sources,
			corev1.VolumeProjection{
				DownwardAPI: &corev1.DownwardAPIProjection{
					Items: []corev1.DownwardAPIVolumeFile{
						{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: fmt.Sprintf("metadata.annotations['%s']", typeAnnotation),
							},
							Path: "type",
						},
					},
				},
			},
		)
	}
	if b.Spec.Provider != "" {
		providerAnnotation := fmt.Sprintf("%s-provider", key)
		ps.Spec.Template.Annotations[providerAnnotation] = b.Spec.Provider
		volume.VolumeSource.Projected.Sources = append(volume.VolumeSource.Projected.Sources,
			corev1.VolumeProjection{
				DownwardAPI: &corev1.DownwardAPIProjection{
					Items: []corev1.DownwardAPIVolumeFile{
						{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: fmt.Sprintf("metadata.annotations['%s']", providerAnnotation),
							},
							Path: "provider",
						},
					},
				},
			},
		)
	}
	ps.Spec.Template.Spec.Volumes = append(ps.Spec.Template.Spec.Volumes, volume)
	injectedSecrets.Insert(sb.Name)
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
	ps.Annotations[key] = sb.Name

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
	key := b.annotationKey()
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
			if e.Key == "type" && b.Spec.Type != "" {
				typeAnnotation := fmt.Sprintf("%s-type", key)
				c.Env = append(c.Env, corev1.EnvVar{
					Name: e.Name,
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: fmt.Sprintf("metadata.annotations['%s']", typeAnnotation),
						},
					},
				})
				continue
			}
			if e.Key == "provider" && b.Spec.Provider != "" {
				providerAnnotation := fmt.Sprintf("%s-provider", key)
				c.Env = append(c.Env, corev1.EnvVar{
					Name: e.Name,
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: fmt.Sprintf("metadata.annotations['%s']", providerAnnotation),
						},
					},
				})
				continue
			}
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
			if !b.isInjectedEnv(iv, allInjectedSecrets) || !b.isInjectedEnv(jv, allInjectedSecrets) {
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
		if c.Name == t {
			return true
		}
	}
	return false
}

func (b *ServiceBindingProjection) Undo(ctx context.Context, ps *duckv1.WithPod) {
	if ps.Annotations == nil {
		ps.Annotations = map[string]string{}
	}
	if ps.Spec.Template.Annotations == nil {
		ps.Spec.Template.Annotations = map[string]string{}
	}

	key := b.annotationKey()
	removeSecrets := sets.NewString(ps.Annotations[key], b.Spec.Binding.Name)
	removeVolumes := sets.NewString()
	delete(ps.Annotations, key)
	delete(ps.Spec.Template.Annotations, fmt.Sprintf("%s-type", key))
	delete(ps.Spec.Template.Annotations, fmt.Sprintf("%s-provider", key))

	preservedVolumes := []corev1.Volume{}
	for _, v := range ps.Spec.Template.Spec.Volumes {
		if v.Projected != nil && len(v.Projected.Sources) > 0 &&
			v.Projected.Sources[0].Secret != nil &&
			removeSecrets.Has(v.Projected.Sources[0].Secret.Name) {
			removeVolumes.Insert(v.Name)
			continue
		}
		preservedVolumes = append(preservedVolumes, v)
	}
	ps.Spec.Template.Spec.Volumes = preservedVolumes

	for i := range ps.Spec.Template.Spec.InitContainers {
		b.undoContainer(ctx, ps, &ps.Spec.Template.Spec.InitContainers[i], removeSecrets, removeVolumes)
	}
	for i := range ps.Spec.Template.Spec.Containers {
		b.undoContainer(ctx, ps, &ps.Spec.Template.Spec.Containers[i], removeSecrets, removeVolumes)
	}
}

func (b *ServiceBindingProjection) undoContainer(ctx context.Context, ps *duckv1.WithPod, c *corev1.Container, removeSecrets, removeVolumes sets.String) {
	preservedMounts := []corev1.VolumeMount{}
	for _, vm := range c.VolumeMounts {
		if !removeVolumes.Has(vm.Name) {
			preservedMounts = append(preservedMounts, vm)
		}
	}
	c.VolumeMounts = preservedMounts

	preservedEnv := []corev1.EnvVar{}
	for _, e := range c.Env {
		if !b.isInjectedEnv(e, removeSecrets) {
			preservedEnv = append(preservedEnv, e)
		}
	}
	c.Env = preservedEnv
}

func (b *ServiceBindingProjection) annotationKey() string {
	return fmt.Sprintf("%s-%x", ServiceBindingProjectionAnnotationKey, sha1.Sum([]byte(b.Name)))
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
		if v.Projected != nil && len(v.Projected.Sources) > 0 &&
			v.Projected.Sources[0].Secret != nil &&
			secrets.Has(v.Projected.Sources[0].Secret.Name) {
			volumes.Insert(v.Name)
		}
	}
	return secrets, volumes
}

var fieldPathAnnotationRe = regexp.MustCompile(fmt.Sprintf(`^%s[0-9a-f]+%s(type|provider)%s$`, regexp.QuoteMeta(fmt.Sprintf("metadata.annotations['%s-", ServiceBindingProjectionAnnotationKey)), "-", "']"))

func (b *ServiceBindingProjection) isInjectedEnv(e corev1.EnvVar, allInjectedSecrets sets.String) bool {
	if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil && allInjectedSecrets.Has(e.ValueFrom.SecretKeyRef.Name) {
		return true
	}
	if e.ValueFrom != nil && e.ValueFrom.FieldRef != nil && fieldPathAnnotationRe.MatchString(e.ValueFrom.FieldRef.FieldPath) {
		return true
	}
	return false
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
