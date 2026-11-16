#!/bin/bash
#
# Script to access the argo mysql database for debugging
#
set -o errexit
set -o nounset
set -o pipefail


kubectl --namespace mysql exec -it service/mysql -- mysql -u root -proot argo
