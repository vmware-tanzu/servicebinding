#!/usr/bin/env bash

# Copyright 2020 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

readonly REPO_ROOT_DIR="$(git rev-parse --show-toplevel)"
readonly TMP_DIFFROOT="$(mktemp -d -p ${REPO_ROOT_DIR})"

cleanup() {
  rm -rf "${TMP_DIFFROOT}"
}

trap "cleanup" EXIT SIGINT

cleanup

# Save working tree state
mkdir -p "${TMP_DIFFROOT}/pkg"
cp -aR "${REPO_ROOT_DIR}/Gopkg.lock" "${REPO_ROOT_DIR}/pkg" "${REPO_ROOT_DIR}/vendor" "${TMP_DIFFROOT}"

# TODO(mattmoor): We should be able to rm -rf pkg/client/ and vendor/

"${REPO_ROOT_DIR}/hack/update-codegen.sh"
echo "Diffing ${REPO_ROOT_DIR} against freshly generated codegen"
ret=0
diff -Naupr "${REPO_ROOT_DIR}/pkg" "${TMP_DIFFROOT}/pkg" || ret=1
diff -Naupr --no-dereference "${REPO_ROOT_DIR}/vendor" "${TMP_DIFFROOT}/vendor" || ret=1

# Restore working tree state
rm -fr "${TMP_DIFFROOT}/config"
rm -fr "${REPO_ROOT_DIR}/Gopkg.lock" "${REPO_ROOT_DIR}/pkg" "${REPO_ROOT_DIR}/vendor"
cp -aR "${TMP_DIFFROOT}"/* "${REPO_ROOT_DIR}"

if [[ $ret -eq 0 ]]
then
  echo "${REPO_ROOT_DIR} up to date."
else
  echo "ERROR: ${REPO_ROOT_DIR} is out of date. Please run ./hack/update-codegen.sh"
  exit 1
fi
