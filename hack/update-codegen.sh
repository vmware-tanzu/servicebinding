#!/usr/bin/env bash

# Copyright 2020 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

export GO111MODULE=on
# If we run with -mod=vendor here, then generate-groups.sh looks for vendor files in the wrong place.
export GOFLAGS=-mod=

REPO_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${REPO_ROOT}; ls -d -1 $(dirname ${BASH_SOURCE})/../vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

KNATIVE_CODEGEN_PKG=${KNATIVE_CODEGEN_PKG:-$(cd ${REPO_ROOT}; ls -d -1 $(dirname ${BASH_SOURCE})/../vendor/knative.dev/pkg 2>/dev/null || echo ../pkg)}

chmod +x ${CODEGEN_PKG}/generate-groups.sh
chmod +x ${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh

API_GROUPS="bindings:v1alpha1 duck:v1alpha1 service:v1alpha2"

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/vmware-labs/service-bindings/pkg/client github.com/vmware-labs/service-bindings/pkg/apis \
  "$API_GROUPS" \
  --go-header-file ${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt

# Knative Injection
${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh "injection" \
  github.com/vmware-labs/service-bindings/pkg/client github.com/vmware-labs/service-bindings/pkg/apis \
  "$API_GROUPS" \
  --go-header-file ${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt

# Make sure our dependencies are up-to-date
${REPO_ROOT}/hack/update-deps.sh
