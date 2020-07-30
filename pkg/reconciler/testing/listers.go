/*
Copyright 2020 VMware, Inc.
Portions Copyright 2020 The Knative Authors.
SPDX-License-Identifier: Apache-2.0
*/

// modeled after https://github.com/knative/serving/blob/v0.16.0/pkg/reconciler/testing/v1/listers.go

package testing

import (
	labsv1alpha1 "github.com/vmware-labs/service-bindings/pkg/apis/labs/v1alpha1"
	servicebindingv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebinding/v1alpha2"
	servicebindinginternalv1alpha2 "github.com/vmware-labs/service-bindings/pkg/apis/servicebindinginternal/v1alpha2"
	fakeservicebindingsclientset "github.com/vmware-labs/service-bindings/pkg/client/clientset/versioned/fake"
	labslisters "github.com/vmware-labs/service-bindings/pkg/client/listers/labs/v1alpha1"
	servicebindinglisters "github.com/vmware-labs/service-bindings/pkg/client/listers/servicebinding/v1alpha2"
	servicebindinginternallisters "github.com/vmware-labs/service-bindings/pkg/client/listers/servicebindinginternal/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/reconciler/testing"
)

var clientSetSchemes = []func(*runtime.Scheme) error{
	fakekubeclientset.AddToScheme,
	fakeservicebindingsclientset.AddToScheme,
}

type Listers struct {
	sorter testing.ObjectSorter
}

func NewListers(objs []runtime.Object) Listers {
	scheme := NewScheme()

	ls := Listers{
		sorter: testing.NewObjectSorter(scheme),
	}

	ls.sorter.AddObjects(objs...)

	return ls
}

func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	for _, addTo := range clientSetSchemes {
		addTo(scheme)
	}
	return scheme
}

func (*Listers) NewScheme() *runtime.Scheme {
	return NewScheme()
}

func (l *Listers) IndexerFor(obj runtime.Object) cache.Indexer {
	return l.sorter.IndexerForObjectType(obj)
}

func (l *Listers) GetKubeObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakekubeclientset.AddToScheme)
}

func (l *Listers) GetServiceBindingsObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakeservicebindingsclientset.AddToScheme)
}

func (l *Listers) GetSecretLister() corev1listers.SecretLister {
	return corev1listers.NewSecretLister(l.IndexerFor(&corev1.Secret{}))
}

func (l *Listers) GetServiceBindingLister() servicebindinglisters.ServiceBindingLister {
	return servicebindinglisters.NewServiceBindingLister(l.IndexerFor(&servicebindingv1alpha2.ServiceBinding{}))
}

func (l *Listers) GetServiceBindingProjectionLister() servicebindinginternallisters.ServiceBindingProjectionLister {
	return servicebindinginternallisters.NewServiceBindingProjectionLister(l.IndexerFor(&servicebindinginternalv1alpha2.ServiceBindingProjection{}))
}

func (l *Listers) GetProvisionedServiceLister() labslisters.ProvisionedServiceLister {
	return labslisters.NewProvisionedServiceLister(l.IndexerFor(&labsv1alpha1.ProvisionedService{}))
}
