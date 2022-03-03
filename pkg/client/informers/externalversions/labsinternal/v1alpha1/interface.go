/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "github.com/vmware-tanzu/servicebinding/pkg/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// ServiceBindingProjections returns a ServiceBindingProjectionInformer.
	ServiceBindingProjections() ServiceBindingProjectionInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// ServiceBindingProjections returns a ServiceBindingProjectionInformer.
func (v *version) ServiceBindingProjections() ServiceBindingProjectionInformer {
	return &serviceBindingProjectionInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
