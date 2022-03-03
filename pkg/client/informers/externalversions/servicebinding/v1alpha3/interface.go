/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha3

import (
	internalinterfaces "github.com/vmware-tanzu/servicebinding/pkg/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// ServiceBindings returns a ServiceBindingInformer.
	ServiceBindings() ServiceBindingInformer
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

// ServiceBindings returns a ServiceBindingInformer.
func (v *version) ServiceBindings() ServiceBindingInformer {
	return &serviceBindingInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
