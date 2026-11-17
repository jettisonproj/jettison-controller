#!/bin/bash
#
# Applies the `telepresence leave` command for the jettison-controller
#
set -o errexit
set -o nounset
set -o pipefail

cd "$(dirname "$0")"
source ./common.sh
cd ../..

echo "Setting context to ${NAMESPACE}"
kubectl config set-context --current --namespace "${NAMESPACE}"

REPLICA_NAME="$(
  kubectl get replicaset -o json | \
    jq -r '.items[] | select((.metadata.name | startswith("'"${REPLICA_PREFIX}"'")) and .spec.replicas == '"${EXPECTED_REPLICAS}"') | .metadata.name'
)"

if [[ -z "${REPLICA_NAME}" ]]; then
  echo "FAIL: did not find replica with prefix ${REPLICA_PREFIX} and expected replicas ${EXPECTED_REPLICAS}"
  echo "todo: need to handle 2 replicas. May need to scale down to 1 temporarily"
  exit 1
fi

echo "running leave for replica: ${REPLICA_NAME}"
telepresence leave "${REPLICA_NAME}"
echo "left the replica: ${REPLICA_NAME}"
