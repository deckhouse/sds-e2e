#---
#apiVersion: deckhouse.io/v1
#kind: User
#metadata:
#  name: admin
#spec:
#  email: admin@cluster.local
#  password: JDJ5JDEwJGxTSHpOU25EOXlYUUY1UlVTRTB2V09zektSU29hdGhjOC5XY1o0ckEvblZZL0lvRGdVSmIyCgo=
#---
#apiVersion: deckhouse.io/v1alpha1
#kind: Group
#metadata:
#  name: admins
#spec:
#  name: admins
#  members:
#    - kind: User
#      name: admin
---
apiVersion: deckhouse.io/v1
kind: NodeGroup
metadata:
  name: worker
spec:
  disruptions:
    approvalMode: Automatic
  kubelet:
    containerLogMaxFiles: 4
    containerLogMaxSize: 50Mi
    maxPods: 200
    resourceReservation:
      mode: "Off"
  nodeType: Static
---
apiVersion: deckhouse.io/v1alpha1
kind: ModulePullOverride
metadata:
  name: sds-local-volume
spec:
  imageTag: main
  scanInterval: 15s
  source: deckhouse
---
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: sds-local-volume
spec:
  enabled: true
---
apiVersion: deckhouse.io/v1alpha1
kind: ModulePullOverride
metadata:
  name: sds-replicated-volume
spec:
  imageTag: main
  scanInterval: 15s
  source: deckhouse
---
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: sds-replicated-volume
spec:
  enabled: true
---
apiVersion: deckhouse.io/v1alpha1
kind: ModulePullOverride
metadata:
  name: sds-node-configurator
spec:
  imageTag: main
  scanInterval: 15s
  source: deckhouse
---
apiVersion: deckhouse.io/v1alpha1
kind: ModuleConfig
metadata:
  name: sds-node-configurator
spec:
  enabled: true
apiVersion: deckhouse.io/v1alpha1
---
apiVersion: deckhouse.io/v1alpha1
kind: ModuleSource
metadata:
  name: deckhouse
  annotations:
    meta.helm.sh/release-name: deckhouse
    meta.helm.sh/release-namespace: d8-system
  labels:
    app.kubernetes.io/managed-by: Helm
spec:
  registry:
    ca: ""
    dockerCfg: %s
    repo: dev-registry.deckhouse.io/sys/deckhouse-oss/modules
    scheme: HTTPS
  releaseChannel: ""
