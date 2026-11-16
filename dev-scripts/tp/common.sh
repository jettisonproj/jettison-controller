#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail


NAMESPACE="jettisonproj"
REPLICA_PREFIX="jettison-controller-manager-"
EXPECTED_REPLICAS=1
