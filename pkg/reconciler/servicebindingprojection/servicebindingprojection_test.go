/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package servicebindingprojection

import (
	"testing"

	"knative.dev/pkg/configmap"

	// register injection fakes
	_ "github.com/vmware-labs/service-bindings/pkg/client/injection/informers/servicebindinginternal/v1alpha2/servicebindingprojection/fake"
	_ "knative.dev/pkg/client/injection/ducks/duck/v1/podspecable/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/namespace/fake"
	_ "knative.dev/pkg/injection/clients/dynamicclient/fake"

	. "knative.dev/pkg/reconciler/testing"
)

func TestNewController(t *testing.T) {
	ctx, _ := SetupFakeContext(t)

	c := NewController(ctx, configmap.NewStaticWatcher())

	if c == nil {
		t.Fatal("expected NewController to return a non-nil value")
	}
}
