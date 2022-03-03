/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"

	v1alpha1 "github.com/vmware-tanzu/servicebinding/pkg/apis/labs/v1alpha1"
	labsinternalv1alpha1 "github.com/vmware-tanzu/servicebinding/pkg/apis/labsinternal/v1alpha1"
	v1alpha3 "github.com/vmware-tanzu/servicebinding/pkg/apis/servicebinding/v1alpha3"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=bindings.labs.vmware.com, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithResource("provisionedservices"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Bindings().V1alpha1().ProvisionedServices().Informer()}, nil

		// Group=internal.bindings.labs.vmware.com, Version=v1alpha1
	case labsinternalv1alpha1.SchemeGroupVersion.WithResource("servicebindingprojections"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Internal().V1alpha1().ServiceBindingProjections().Informer()}, nil

		// Group=servicebinding.io, Version=v1alpha3
	case v1alpha3.SchemeGroupVersion.WithResource("servicebindings"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Servicebinding().V1alpha3().ServiceBindings().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
