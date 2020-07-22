// +build tools

/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package tools

import (
	_ "k8s.io/code-generator"

	// codegen: hack/generate-knative.sh
	_ "knative.dev/pkg/hack"

	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/code-generator/cmd/defaulter-gen"
	_ "k8s.io/code-generator/cmd/informer-gen"
	_ "k8s.io/code-generator/cmd/lister-gen"
	_ "knative.dev/pkg/codegen/cmd/injection-gen"
)
