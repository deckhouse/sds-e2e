#!/bin/bash

# NOTE: this script is used to run tests. All the parameters passed to script are passed to go test.

# Required environment variables (examples):
#
# export licensekey="mdfbweufglkwjbdlkfe"
# export hv_ssh_dst="tfadm@172.17.1.67"
# export e2e_namespace="e2e-username"

# Check required environment variables
missing_vars=()

if [ -z "$hv_ssh_dst" ]; then
    missing_vars+=("hv_ssh_dst")
fi

if [ -z "$e2e_namespace" ]; then
    missing_vars+=("e2e_namespace")
fi

if [ ${#missing_vars[@]} -gt 0 ]; then
    echo "Error: Required environment variables are not set:" >&2
    for var in "${missing_vars[@]}"; do
        echo "  - $var" >&2
    done
    echo "" >&2
    echo "Usage:" >&2
    echo "  export hv_ssh_dst=\"<user>@<host>\"" >&2
    echo "  export e2e_namespace=\"e2e-username\"" >&2
    echo "  export licensekey=\"<EE deckhouse license key>\"" >&2
    echo "" >&2
    echo "Then run: $0 [test-file name and other go test parameters...]" >&2
    exit 1
fi

go test \
  -v \
  -timeout 30m \
  ./tests/tools.go \
  "$@" \
  -debug \
  -sshhost $hv_ssh_dst \
  -hypervisorkconfig kube-hypervisor.config \
  -skipoptional \
  -keepstate \
  -clustertype "Ubuntu 22 mini" \
  -hvstorageclass "hpe" \
  -namespace $e2e_namespace

