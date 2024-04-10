# General cluster parameters.
# https://deckhouse.io/documentation/v1/installing/configuration.html#clusterconfiguration
apiVersion: deckhouse.io/v1
kind: ClusterConfiguration
clusterType: Static
# Address space of the cluster's Pods.
podSubnetCIDR: 10.112.0.0/16
# Address space of the cluster's services.
serviceSubnetCIDR: 10.225.0.0/16
kubernetesVersion: "Automatic"
# Cluster domain (used for local routing).
clusterDomain: "cluster.local"
---
# Settings for the bootstrapping the Deckhouse cluster
# https://deckhouse.io/documentation/v1/installing/configuration.html#initconfiguration
apiVersion: deckhouse.io/v1
kind: InitConfiguration
deckhouse:
  # Address of the Docker registry where the Deckhouse images are located
  imagesRepo: dev-registry.deckhouse.io/sys/deckhouse-oss
  # A special string with your token to access Docker registry (generated automatically for your license token)
  registryDockerCfg: %s
  devBranch: main
---
# Deckhouse module settings.
# https://deckhouse.io/documentation/v1/modules/002-deckhouse/configuration.html
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: deckhouse
spec:
  version: 1
  enabled: true
  settings:
    bundle: Default
    releaseChannel: Stable
    logLevel: Info
---
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: flant-integration
spec:
  enabled: false
  settings:
    kubeall:
      host: fake.host
  version: 1
---
# Global Deckhouse settings.
# https://deckhouse.ru/documentation/v1/deckhouse-configure-global.html#parameters
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: global
spec:
  version: 1
  settings:
    modules:
      publicDomainTemplate: "%%s.virtualmachines.local"
---
# user-authn module settings.
# https://deckhouse.io/documentation/v1/modules/150-user-authn/configuration.html
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: user-authn
spec:
  version: 1
  enabled: true
  settings:
    controlPlaneConfigurator:
      dexCAMode: DoNotNeed
    # Enabling access to the API server through Ingress.
    # https://deckhouse.io/documentation/v1/modules/150-user-authn/configuration.html#parameters-publishapi
    publishAPI:
      enable: true
      https:
        mode: Global
        global:
          kubeconfigGeneratorMasterCA: ""
---
# cni-cilium module settings.
# https://deckhouse.io/documentation/v1/modules/021-cni-cilium/configuration.html
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: cni-cilium
spec:
  version: 1
  # Enable cni-cilium module
  enabled: true
  settings:
    # cni-cilium module settings
    # https://deckhouse.io/documentation/v1/modules/021-cni-cilium/configuration.html
    tunnelMode: VXLAN
---
# Static cluster settings.
# https://deckhouse.io/documentation/v1/installing/configuration.html#staticclusterconfiguration
apiVersion: deckhouse.io/v1
kind: StaticClusterConfiguration
# List of internal cluster networks (e.g., '10.0.4.0/24'), which is
# used for linking Kubernetes components (kube-apiserver, kubelet etc.).
# If every node in cluster has only one network interface
# StaticClusterConfiguration resource can be skipped.
internalNetworkCIDRs:
- 10.10.10.0/24