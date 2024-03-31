#!/bin/bash

kubectl create -f - <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: gitlab-runner-deploy
  namespace: d8-service-accounts
---
apiVersion: v1
kind: Secret
metadata:
  name: gitlab-runner-deploy-token
  namespace: d8-service-accounts
  annotations:
    kubernetes.io/service-account.name: gitlab-runner-deploy
type: kubernetes.io/service-account-token
EOF

kubectl create -f - <<EOF
apiVersion: deckhouse.io/v1
kind: ClusterAuthorizationRule
metadata:
  name: gitlab-runner-deploy
spec:
  subjects:
  - kind: ServiceAccount
    name: gitlab-runner-deploy
    namespace: d8-service-accounts
  accessLevel: SuperAdmin
EOF

export CLUSTER_NAME=my-cluster
export USER_NAME=gitlab-runner-deploy.my-cluster
export CONTEXT_NAME=${CLUSTER_NAME}-${USER_NAME}
export FILE_NAME=kube.config

kubectl get cm kube-root-ca.crt -o jsonpath='{ .data.ca\.crt }' > /tmp/ca.crt

kubectl config set-cluster $CLUSTER_NAME --embed-certs=true \
  --server=https://$(kubectl get ep kubernetes -o json | jq -rc '.subsets[0] | "\(.addresses[0].ip):\(.ports[0].port)"') \
  --certificate-authority=/tmp/ca.crt \
  --kubeconfig=$FILE_NAME

kubectl config set-credentials $USER_NAME \
  --token=$(kubectl -n d8-service-accounts get secret gitlab-runner-deploy-token -o json |jq -r '.data["token"]' | base64 -d) \
  --kubeconfig=$FILE_NAME

kubectl config set-context $CONTEXT_NAME \
  --cluster=$CLUSTER_NAME --user=$USER_NAME \
  --kubeconfig=$FILE_NAME

kubectl config use-context $CONTEXT_NAME --kubeconfig=$FILE_NAME
