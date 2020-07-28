#!/usr/bin/env bash

# Copyright 2020 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
CODEGEN_PKG=${CODEGEN_PKG:-$(go mod download -json 2>/dev/null | jq -r 'select(.Path == "k8s.io/code-generator").Dir')}
KNATIVE_CODEGEN_PKG=${KNATIVE_CODEGEN_PKG:-$(go mod download -json 2>/dev/null | jq -r 'select(.Path == "knative.dev/pkg").Dir')}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf ${TMP_DIR}' EXIT
export GOPATH=${GOPATH:-${TMP_DIR}}

TMP_REPO_PATH="${TMP_DIR}/src/github.com/vmware-labs/service-bindings"
mkdir -p "$(dirname "${TMP_REPO_PATH}")" && ln -s "${REPO_ROOT}" "${TMP_REPO_PATH}"

API_GROUPS="duck:v1alpha1 labs:v1alpha1 servicebinding:v1alpha2 servicebindinginternal:v1alpha2"

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
bash ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/vmware-labs/service-bindings/pkg/client github.com/vmware-labs/service-bindings/pkg/apis \
  "$API_GROUPS" \
  --output-base "${TMP_DIR}/src" \
  --go-header-file ${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt

# Knative Injection
bash ${KNATIVE_CODEGEN_PKG}/hack/generate-knative.sh "injection" \
  github.com/vmware-labs/service-bindings/pkg/client github.com/vmware-labs/service-bindings/pkg/apis \
  "$API_GROUPS" \
  --output-base "${TMP_DIR}/src" \
  --go-header-file ${REPO_ROOT}/hack/boilerplate/boilerplate.go.txt
