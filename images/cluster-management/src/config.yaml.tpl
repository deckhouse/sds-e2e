# General cluster parameters.
# https://deckhouse.io/documentation/v1/installing/configuration.html#clusterconfiguration
apiVersion: deckhouse.io/v1
kind: ClusterConfiguration
clusterType: Static
# Address space of the cluster's Pods.
podSubnetCIDR: 10.112.0.0/16
# Address space of the cluster's services.
serviceSubnetCIDR: 10.223.0.0/16
# Kubernetes version to install.
kubernetesVersion: "1.25"
# Cluster domain (used for local routing).
clusterDomain: "internalcluster.local"
---
# Section for bootstrapping the Deckhouse cluster.
# https://deckhouse.io/documentation/v1/installing/configuration.html#initconfiguration
apiVersion: deckhouse.io/v1
kind: InitConfiguration
deckhouse:
  devBranch: main
  # Address of the Docker registry where the Deckhouse images are located
  imagesRepo: dev-registry.deckhouse.io/sys/deckhouse-oss
  # A special string with your token to access Docker registry (generated automatically for your license token)
  registryDockerCfg: %s
  configOverrides:
    global:
      modules:
        # Template that will be used for system apps domains within the cluster.
        # E.g., Grafana for %s.virtualmachines.local will be available as 'grafana.virtualmachines.local'.
        # You can change it to your own or follow the steps in the guide and change it after installation.
        publicDomainTemplate: "%s.virtualmachines.local"
    userAuthn:
      controlPlaneConfigurator:
        dexCAMode: DoNotNeed
      # Enabling access to the API server through Ingress.
      # https://deckhouse.io/documentation/v1/modules/150-user-authn/configuration.html#parameters-publishapi
      publishAPI:
        enable: true
        https:
          mode: Global
    # Enable cni-cilium module
    cniCiliumEnabled: true
    # cni-cilium module settings
    # https://deckhouse.io/documentation/v1/modules/021-cni-cilium/configuration.html
    cniCilium:
      tunnelMode: VXLAN
---
# Section with the parameters of the static cluster.
# https://deckhouse.io/documentation/v1/installing/configuration.html#staticclusterconfiguration
apiVersion: deckhouse.io/v1
kind: StaticClusterConfiguration
# List of internal cluster networks (e.g., '10.0.4.0/24'), which is
# used for linking Kubernetes components (kube-apiserver, kubelet etc.).
# If every node in cluster has only one network interface
# StaticClusterConfiguration resource can be skipped.
internalNetworkCIDRs:
  - 10.10.10.0/24