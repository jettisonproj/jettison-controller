#!/bin/bash
#
# Applies the `telepresence replace` command for the jettison-controller
#
set -o errexit
set -o nounset
set -o pipefail

cd "$(dirname "$0")"
source ./common.sh
cd ../.. # top level for top level mounts

echo "Setting context to ${NAMESPACE}"
kubectl config set-context --current --namespace "${NAMESPACE}"

if helm status --namespace ambassador traffic-manager; then
  echo "telepresence helm already installed"
else
  echo "installing telepresence..."
  telepresence helm install
  telepresence connect
fi

REPLICA_NAME="$(
  kubectl get replicaset -o json | \
    jq -r '.items[] | select((.metadata.name | startswith("'"${REPLICA_PREFIX}"'")) and .spec.replicas == '"${EXPECTED_REPLICAS}"') | .metadata.name'
)"

if [[ -z "${REPLICA_NAME}" ]]; then
  echo "FAIL: did not find replica with prefix ${REPLICA_PREFIX} and expected replicas ${EXPECTED_REPLICAS}"
  echo "todo: need to handle 2 replicas. May need to scale down to 1 temporarily"
  exit 1
fi

echo "Running replace for replica: ${REPLICA_NAME}"

telepresence \
  replace \
  "${REPLICA_NAME}" \
  --mount tpmount \
  --env-file tpenv \
  --port 2847:2846

echo "waiting for replacement"
kubectl \
  wait \
  --for 'jsonpath=.status.replicas=1' \
  --for 'jsonpath=.status.availableReplicas=1' \
  --for 'jsonpath=.status.terminatingReplicas=0' \
  "replicaset/${REPLICA_NAME}"

echo "replacement is available"
