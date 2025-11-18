#!/bin/bash

export licensekey="$1"
export hv_ssh_dst="tfadm@172.17.1.67"

go test \
  -v \
  -timeout 30m \
  ./tests/tools.go \
  $2 \
  -debug \
  -sshhost $hv_ssh_dst \
  -hypervisorkconfig kube-hypervisor.config \
  -skipoptional \
  -keepstate \
  -clustertype "Ubuntu 22 mini" \
  -hvstorageclass "hpe" \
  -namespace e2e-ayakubov

