/*
Copyright 2020 VMware, Inc.
Portions Copyright 2020 The Knative Authors.
SPDX-License-Identifier: Apache-2.0
*/

// modeled after https://github.com/knative/serving/blob/v0.16.0/pkg/reconciler/testing/v1/factory.go

package testing

import (
	"context"
	"encoding/json"
	"testing"

	labsv1alpha1 "github.com/vmware-tanzu/servicebinding/pkg/apis/labs/v1alpha1"
	fakeservicebindingsclient "github.com/vmware-tanzu/servicebinding/pkg/client/injection/client/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	fakedynamicclient "knative.dev/pkg/injection/clients/dynamicclient/fake"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/reconciler"
	rtesting "knative.dev/pkg/reconciler/testing"
	"knative.dev/pkg/tracker"
)

const (
	// maxEventBufferSize is the estimated max number of event notifications that
	// can be buffered during reconciliation.
	maxEventBufferSize = 10
)

// Ctor functions create a k8s controller with given params.
type Ctor func(context.Context, *Listers, configmap.Watcher) controller.Reconciler

// MakeFactory creates a reconciler factory with fake clients and controller created by `ctor`.
func MakeFactory(ctor Ctor) rtesting.Factory {
	return func(t *testing.T, r *rtesting.TableRow) (
		controller.Reconciler, rtesting.ActionRecorderList, rtesting.EventList) {
		ls := NewListers(r.Objects)

		ctx := r.Ctx
		if ctx == nil {
			ctx = context.Background()
		}
		logger := logtesting.TestLogger(t)
		ctx = logging.WithLogger(ctx, logger)

		ctx, kubeClient := fakekubeclient.With(ctx, ls.GetKubeObjects()...)
		ctx, servicebindingsClient := fakeservicebindingsclient.With(ctx, ls.GetServiceBindingsObjects()...)
		ctx, dynamicClient := fakedynamicclient.With(ctx, ls.NewScheme(), ToUnstructured(t, ls.NewScheme(), r.Objects)...)
		ctx = context.WithValue(ctx, TrackerKey, &rtesting.FakeTracker{})

		// The dynamic client's support for patching is BS.  Implement it
		// here via PrependReactor (this can be overridden below by the
		// provided reactors).
		dynamicClient.PrependReactor("patch", "*",
			func(action ktesting.Action) (bool, runtime.Object, error) {
				return true, nil, nil
			})

		eventRecorder := record.NewFakeRecorder(maxEventBufferSize)
		ctx = controller.WithEventRecorder(ctx, eventRecorder)

		// This is needed for the tests that use generated names and
		// the object cannot be created beforehand.
		kubeClient.PrependReactor("create", "*",
			func(action ktesting.Action) (bool, runtime.Object, error) {
				ca := action.(ktesting.CreateAction)
				ls.IndexerFor(ca.GetObject()).Add(ca.GetObject())
				return false, nil, nil
			},
		)
		// This is needed by controller tests, which use GenerateName
		rtesting.PrependGenerateNameReactor(&servicebindingsClient.Fake)

		// Set up our Controller from the fakes.
		c := ctor(ctx, &ls, configmap.NewStaticWatcher())
		// Update the context with the stuff we decorated it with.
		r.Ctx = ctx

		// The Reconciler won't do any work until it becomes the leader.
		if la, ok := c.(reconciler.LeaderAware); ok {
			la.Promote(reconciler.UniversalBucket(), func(reconciler.Bucket, types.NamespacedName) {})
		}

		for _, reactor := range r.WithReactors {
			kubeClient.PrependReactor("*", "*", reactor)
			servicebindingsClient.PrependReactor("*", "*", reactor)
			dynamicClient.PrependReactor("*", "*", reactor)
		}

		// Validate all Create operations through the servicebindings client.
		servicebindingsClient.PrependReactor("create", "*", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			return rtesting.ValidateCreates(context.Background(), action)
		})
		servicebindingsClient.PrependReactor("update", "*", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			return rtesting.ValidateUpdates(context.Background(), action)
		})

		actionRecorderList := rtesting.ActionRecorderList{dynamicClient, servicebindingsClient, kubeClient}
		eventList := rtesting.EventList{Recorder: eventRecorder}

		return c, actionRecorderList, eventList
	}
}

// ToUnstructured takes a list of k8s resources and converts them to
// Unstructured objects.
// We must pass objects as Unstructured to the dynamic client fake, or it
// won't handle them properly.
func ToUnstructured(t *testing.T, sch *runtime.Scheme, objs []runtime.Object) (us []runtime.Object) {
	for _, obj := range objs {
		obj = obj.DeepCopyObject() // Don't mess with the primary copy
		// Determine and set the TypeMeta for this object based on our test scheme.
		gvks, _, err := sch.ObjectKinds(obj)
		if err != nil {
			t.Fatal("Unable to determine kind for type:", err)
		}
		apiv, k := gvks[0].ToAPIVersionAndKind()
		ta, err := meta.TypeAccessor(obj)
		if err != nil {
			t.Fatal("Unable to create type accessor:", err)
		}
		ta.SetAPIVersion(apiv)
		ta.SetKind(k)

		b, err := json.Marshal(obj)
		if err != nil {
			t.Fatal("Unable to marshal:", err)
		}
		u := &unstructured.Unstructured{}
		if err := json.Unmarshal(b, u); err != nil {
			t.Fatal("Unable to unmarshal:", err)
		}
		us = append(us, u)
	}
	return
}

type key struct{}

// TrackerKey is used to looking a FakeTracker in a context.Context
var TrackerKey key = struct{}{}

func GetTracker(ctx context.Context) tracker.Interface {
	return ctx.Value(TrackerKey).(tracker.Interface)
}

// AssertTrackingProvsisionedService will ensure the provided ProvisionedService is being tracked
func AssertTrackingProvisionedService(namespace, name string) func(*testing.T, *rtesting.TableRow) {
	gvk := labsv1alpha1.SchemeGroupVersion.WithKind("ProvisionedService")
	return AssertTrackingObject(gvk, namespace, name)
}

// AssertTrackingSecret will ensure the provided Secret is being tracked
func AssertTrackingSecret(namespace, name string) func(*testing.T, *rtesting.TableRow) {
	gvk := corev1.SchemeGroupVersion.WithKind("Secret")
	return AssertTrackingObject(gvk, namespace, name)
}

// AssertTrackingObject will ensure the following objects are being tracked
func AssertTrackingObject(gvk schema.GroupVersionKind, namespace, name string) func(*testing.T, *rtesting.TableRow) {
	apiVersion, kind := gvk.ToAPIVersionAndKind()

	return func(t *testing.T, r *rtesting.TableRow) {
		tracker := r.Ctx.Value(TrackerKey).(*rtesting.FakeTracker)
		refs := tracker.References()

		for _, ref := range refs {
			if ref.APIVersion == apiVersion &&
				ref.Name == name &&
				ref.Namespace == namespace &&
				ref.Kind == kind {
				return
			}
		}

		t.Errorf("Object was not tracked - %s, Name=%s, Namespace=%s", gvk.String(), name, namespace)
	}

}
