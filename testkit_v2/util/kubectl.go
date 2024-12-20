package integration

import (
//	"fmt"
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
//	"k8s.io/apimachinery/pkg/runtime/schema"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var c ctrlruntimeclient.Client

func RunPod(ctx context.Context, cl ctrlruntimeclient.Client) {
//	ctx := context.Background()
	nodeName := "d-shipkov-worker-0"
//	nodeSelector := fmt.Sprintf("\"nodeSelector\": { \"kubernetes.io/hostname\": \"%s\" },", nodeName)
//	pod := &corev1.Pod{}

	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "test1", Name: "bar"}}
	binding := &corev1.Binding{Target: corev1.ObjectReference{Name: nodeName}}
	//c.SubResourceClient("binding").Create(ctx, pod, binding)
	cl.SubResource("binding").Create(ctx, pod, binding)

/*
P O D: &Pod{
  ObjectMeta:{
    Name: controller-nginx-hfn2t
    GenerateName: controller-nginx-
    Namespace: d8-ingress-nginx
    SelfLink:  
    UID: 736752d2-1dcb-43f7-a974-e77b3463c533
    ResourceVersion: 19360872
    Generation: 0
    CreationTimestamp: 2024-12-04 16:02:28 +0300 MSK
    DeletionTimestamp: <nil>
    DeletionGracePeriodSeconds: <nil>
    Labels map[app:controller controller-revision-hash:5669bfc555 name:nginx]
    Annotations: map[vpaObservedContainers:controller, protobuf-exporter, kube-rbac-proxy vpaUpdates:Pod resources updated by controller-nginx: container 0: cpu request, memory request; container 1: cpu request, memory request; container 2: memory request, cpu request]
    OwnerReferences: [{apps.kruise.io/v1alpha1 DaemonSet controller-nginx ba9f69e0-a104-442a-9362-8aee8d3da964 0xc0006dc549 0xc0006dc54a}]
    Finalizers: []
    ManagedFields: [{kruise-manager Update v1 2024-12-04 16:02:28 +0300 MSK FieldsV1 {"f:metadata":{"f:generateName":{},"f:labels":{".":{},"f:app":{},"f:controller-revision-hash":{},"f:name":{}},"f:ownerReferences":{".":{},"k:{\"uid\":\"ba9f69e0-a104-442a-9362-8aee8d3da964\"}":{}}},"f:spec":{"f:affinity":{".":{},"f:nodeAffinity":{".":{},"f:requiredDuringSchedulingIgnoredDuringExecution":{}}},"f:containers":{"k:{\"name\":\"controller\"}":{".":{},"f:args":{},"f:env":{".":{},"k:{\"name\":\"POD_IP\"}":{".":{},"f:name":{},"f:value":{}},"k:{\"name\":\"POD_NAME\"}":{".":{},"f:name":{},"f:valueFrom":{".":{},"f:fieldRef":{}}},"k:{\"name\":\"POD_NAMESPACE\"}":{".":{},"f:name":{},"f:valueFrom":{".":{},"f:fieldRef":{}}}},"f:image":{},"f:imagePullPolicy":{},"f:lifecycle":{".":{},"f:preStop":{".":{},"f:exec":{".":{},"f:command":{}}}},"f:livenessProbe":{".":{},"f:failureThreshold":{},"f:httpGet":{".":{},"f:path":{},"f:port":{},"f:scheme":{}},"f:initialDelaySeconds":{},"f:periodSeconds":{},"f:successThreshold":{},"f:timeoutSeconds":{}},"f:name":{},"f:ports":{".":{},"k:{\"containerPort\":80,\"protocol\":\"TCP\"}":{".":{},"f:containerPort":{},"f:protocol":{}},"k:{\"containerPort\":443,\"protocol\":\"TCP\"}":{".":{},"f:containerPort":{},"f:protocol":{}},"k:{\"containerPort\":443,\"protocol\":\"UDP\"}":{".":{},"f:containerPort":{},"f:protocol":{}},"k:{\"containerPort\":8443,\"protocol\":\"TCP\"}":{".":{},"f:containerPort":{},"f:name":{},"f:protocol":{}}},"f:readinessProbe":{".":{},"f:failureThreshold":{},"f:httpGet":{".":{},"f:path":{},"f:port":{},"f:scheme":{}},"f:initialDelaySeconds":{},"f:periodSeconds":{},"f:successThreshold":{},"f:timeoutSeconds":{}},"f:resources":{".":{},"f:requests":{".":{},"f:ephemeral-storage":{}}},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{},"f:volumeMounts":{".":{},"k:{\"mountPath\":\"/chroot/etc/nginx/ssl/\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/etc/nginx/webhook-ssl/\"}":{".":{},"f:mountPath":{},"f:name":{},"f:readOnly":{}},"k:{\"mountPath\":\"/chroot/tmp/nginx/\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/var/lib/nginx/body\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/var/lib/nginx/fastcgi\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/var/lib/nginx/proxy\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/var/lib/nginx/scgi\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/var/lib/nginx/uwsgi\"}":{".":{},"f:mountPath":{},"f:name":{}}}},"k:{\"name\":\"kube-rbac-proxy\"}":{".":{},"f:args":{},"f:env":{".":{},"k:{\"name\":\"KUBE_RBAC_PROXY_CONFIG\"}":{".":{},"f:name":{},"f:value":{}},"k:{\"name\":\"KUBE_RBAC_PROXY_LISTEN_ADDRESS\"}":{".":{},"f:name":{},"f:valueFrom":{".":{},"f:fieldRef":{}}}},"f:image":{},"f:imagePullPolicy":{},"f:lifecycle":{".":{},"f:preStop":{".":{},"f:exec":{".":{},"f:command":{}}}},"f:name":{},"f:ports":{".":{},"k:{\"containerPort\":4207,\"protocol\":\"TCP\"}":{".":{},"f:containerPort":{},"f:name":{},"f:protocol":{}}},"f:resources":{".":{},"f:requests":{".":{},"f:cpu":{},"f:ephemeral-storage":{},"f:memory":{}}},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{}},"k:{\"name\":\"protobuf-exporter\"}":{".":{},"f:image":{},"f:imagePullPolicy":{},"f:name":{},"f:resources":{".":{},"f:requests":{".":{},"f:cpu":{},"f:ephemeral-storage":{},"f:memory":{}}},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{},"f:volumeMounts":{".":{},"k:{\"mountPath\":\"/var/files\"}":{".":{},"f:mountPath":{},"f:name":{}}}}},"f:dnsPolicy":{},"f:enableServiceLinks":{},"f:imagePullSecrets":{".":{},"k:{\"name\":\"deckhouse-registry\"}":{}},"f:nodeSelector":{},"f:priorityClassName":{},"f:restartPolicy":{},"f:schedulerName":{},"f:securityContext":{},"f:serviceAccount":{},"f:serviceAccountName":{},"f:terminationGracePeriodSeconds":{},"f:tolerations":{},"f:volumes":{".":{},"k:{\"name\":\"client-body-temp-path\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"fastcgi-temp-path\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"proxy-temp-path\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"scgi-temp-path\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"secret-nginx-auth-tls\"}":{".":{},"f:name":{},"f:secret":{".":{},"f:defaultMode":{},"f:secretName":{}}},"k:{\"name\":\"telemetry-config-file\"}":{".":{},"f:configMap":{".":{},"f:defaultMode":{},"f:name":{}},"f:name":{}},"k:{\"name\":\"tmp-nginx\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"uwsgi-temp-path\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"webhook-cert\"}":{".":{},"f:name":{},"f:secret":{".":{},"f:defaultMode":{},"f:secretName":{}}}}}} } {kubelet Update v1 2024-12-04 16:36:49 +0300 MSK FieldsV1 {"f:status":{"f:conditions":{"k:{\"type\":\"ContainersReady\"}":{".":{},"f:lastProbeTime":{},"f:lastTransitionTime":{},"f:status":{},"f:type":{}},"k:{\"type\":\"Initialized\"}":{".":{},"f:lastProbeTime":{},"f:lastTransitionTime":{},"f:status":{},"f:type":{}},"k:{\"type\":\"PodReadyToStartContainers\"}":{".":{},"f:lastProbeTime":{},"f:lastTransitionTime":{},"f:status":{},"f:type":{}},"k:{\"type\":\"Ready\"}":{".":{},"f:lastProbeTime":{},"f:lastTransitionTime":{},"f:status":{},"f:type":{}}},"f:containerStatuses":{},"f:hostIP":{},"f:hostIPs":{},"f:phase":{},"f:podIP":{},"f:podIPs":{".":{},"k:{\"ip\":\"10.111.2.247\"}":{".":{},"f:ip":{}}},"f:startTime":{}}} status}]},
  Spec:PodSpec{
    Volumes:[]Volume{Volume{Name:tmp-nginx,VolumeSource:VolumeSource{HostPath:nil,EmptyDir:&EmptyDirVolumeSource{Medium:,SizeLimit:<nil>,},GCEPersistentDisk:nil,AWSElasticBlockStore:nil,GitRepo:nil,Secret:nil,NFS:nil,ISCSI:nil,Glusterfs:nil,PersistentVolumeClaim:nil,RBD:nil,FlexVolume:nil,Cinder:nil,CephFS:nil,Flocker:nil,DownwardAPI:nil,FC:nil,AzureFile:nil,ConfigMap:nil,VsphereVolume:nil,Quobyte:nil,AzureDisk:nil,PhotonPersistentDisk:nil,PortworxVolume:nil,ScaleIO:nil,Projected:nil,StorageOS:nil,CSI:nil,Ephemeral:nil,Image:nil,},},Volume{Name:client-body-temp-path,VolumeSource:VolumeSource{HostPath:nil,EmptyDir:&EmptyDirVolumeSource{Medium:,SizeLimit:<nil>,},GCEPersistentDisk:nil,AWSElasticBlockStore:nil,GitRepo:nil,Secret:nil,NFS:nil,ISCSI:nil,Glusterfs:nil,PersistentVolumeClaim:nil,RBD:nil,FlexVolume:nil,Cinder:nil,CephFS:nil,Flocker:nil,DownwardAPI:nil,FC:nil,AzureFile:nil,ConfigMap:nil,VsphereVolume:nil,Quobyte:nil,AzureDisk:nil,PhotonPersistentDisk:nil,PortworxVolume:nil,ScaleIO:nil,Projected:nil,StorageOS:nil,CSI:nil,Ephemeral:nil,Image:nil,},},Volume{Name:fastcgi-temp-path,VolumeSource:VolumeSource{HostPath:nil,EmptyDir:&EmptyDirVolumeSource{Medium:,SizeLimit:<nil>,},GCEPersistentDisk:nil,AWSElasticBlockStore:nil,GitRepo:nil,Secret:nil,NFS:nil,ISCSI:nil,Glusterfs:nil,PersistentVolumeClaim:nil,RBD:nil,FlexVolume:nil,Cinder:nil,CephFS:nil,Flocker:nil,DownwardAPI:nil,FC:nil,AzureFile:nil,ConfigMap:nil,VsphereVolume:nil,Quobyte:nil,AzureDisk:nil,PhotonPersistentDisk:nil,PortworxVolume:nil,ScaleIO:nil,Projected:nil,StorageOS:nil,CSI:nil,Ephemeral:nil,Image:nil,},},Volume{Name:proxy-temp-path,VolumeSource:VolumeSource{HostPath:nil,EmptyDir:&EmptyDirVolumeSource{Medium:,SizeLimit:<nil>,},GCEPersistentDisk:nil,AWSElasticBlockStore:nil,GitRepo:nil,Secret:nil,NFS:nil,ISCSI:nil,Glusterfs:nil,PersistentVolumeClaim:nil,RBD:nil,FlexVolume:nil,Cinder:nil,CephFS:nil,Flocker:nil,DownwardAPI:nil,FC:nil,AzureFile:nil,ConfigMap:nil,VsphereVolume:nil,Quobyte:nil,AzureDisk:nil,PhotonPersistentDisk:nil,PortworxVolume:nil,ScaleIO:nil,Projected:nil,StorageOS:nil,CSI:nil,Ephemeral:nil,Image:nil,},},Volume{Name:scgi-temp-path,VolumeSource:VolumeSource{HostPath:nil,EmptyDir:&EmptyDirVolumeSource{Medium:,SizeLimit:<nil>,},GCEPersistentDisk:nil,AWSElasticBlockStore:nil,GitRepo:nil,Secret:nil,NFS:nil,ISCSI:nil,Glusterfs:nil,PersistentVolumeClaim:nil,RBD:nil,FlexVolume:nil,Cinder:nil,CephFS:nil,Flocker:nil,DownwardAPI:nil,FC:nil,AzureFile:nil,ConfigMap:nil,VsphereVolume:nil,Quobyte:nil,AzureDisk:nil,PhotonPersistentDisk:nil,PortworxVolume:nil,ScaleIO:nil,Projected:nil,StorageOS:nil,CSI:nil,Ephemeral:nil,Image:nil,},},Volume{Name:uwsgi-temp-path,VolumeSource:VolumeSource{HostPath:nil,EmptyDir:&EmptyDirVolumeSource{Medium:,SizeLimit:<nil>,},GCEPersistentDisk:nil,AWSElasticBlockStore:nil,GitRepo:nil,Secret:nil,NFS:nil,ISCSI:nil,Glusterfs:nil,PersistentVolumeClaim:nil,RBD:nil,FlexVolume:nil,Cinder:nil,CephFS:nil,Flocker:nil,DownwardAPI:nil,FC:nil,AzureFile:nil,ConfigMap:nil,VsphereVolume:nil,Quobyte:nil,AzureDisk:nil,PhotonPersistentDisk:nil,PortworxVolume:nil,ScaleIO:nil,Projected:nil,StorageOS:nil,CSI:nil,Ephemeral:nil,Image:nil,},},Volume{Name:secret-nginx-auth-tls,VolumeSource:VolumeSource{HostPath:nil,EmptyDir:nil,GCEPersistentDisk:nil,AWSElasticBlockStore:nil,GitRepo:nil,Secret:&SecretVolumeSource{SecretName:ingress-nginx-nginx-auth-tls,Items:[]KeyToPath{},DefaultMode:*420,Optional:nil,},NFS:nil,ISCSI:nil,Glusterfs:nil,PersistentVolumeClaim:nil,RBD:nil,FlexVolume:nil,Cinder:nil,CephFS:nil,Flocker:nil,DownwardAPI:nil,FC:nil,AzureFile:nil,ConfigMap:nil,VsphereVolume:nil,Quobyte:nil,AzureDisk:nil,PhotonPersistentDisk:nil,PortworxVolume:nil,ScaleIO:nil,Projected:nil,StorageOS:nil,CSI:nil,Ephemeral:nil,Image:nil,},},Volume{Name:webhook-cert,VolumeSource:VolumeSource{HostPath:nil,EmptyDir:nil,GCEPersistentDisk:nil,AWSElasticBlockStore:nil,GitRepo:nil,Secret:&SecretVolumeSource{SecretName:ingress-admission-certificate,Items:[]KeyToPath{},DefaultMode:*420,Optional:nil,},NFS:nil,ISCSI:nil,Glusterfs:nil,PersistentVolumeClaim:nil,RBD:nil,FlexVolume:nil,Cinder:nil,CephFS:nil,Flocker:nil,DownwardAPI:nil,FC:nil,AzureFile:nil,ConfigMap:nil,VsphereVolume:nil,Quobyte:nil,AzureDisk:nil,PhotonPersistentDisk:nil,PortworxVolume:nil,ScaleIO:nil,Projected:nil,StorageOS:nil,CSI:nil,Ephemeral:nil,Image:nil,},},Volume{Name:telemetry-config-file,VolumeSource:VolumeSource{HostPath:nil,EmptyDir:nil,GCEPersistentDisk:nil,AWSElasticBlockStore:nil,GitRepo:nil,Secret:nil,NFS:nil,ISCSI:nil,Glusterfs:nil,PersistentVolumeClaim:nil,RBD:nil,FlexVolume:nil,Cinder:nil,CephFS:nil,Flocker:nil,DownwardAPI:nil,FC:nil,AzureFile:nil,ConfigMap:&ConfigMapVolumeSource{LocalObjectReference:LocalObjectReference{Name:d8-ingress-telemetry-config,},Items:[]KeyToPath{},DefaultMode:*420,Optional:nil,},VsphereVolume:nil,Quobyte:nil,AzureDisk:nil,PhotonPersistentDisk:nil,PortworxVolume:nil,ScaleIO:nil,Projected:nil,StorageOS:nil,CSI:nil,Ephemeral:nil,Image:nil,},},Volume{Name:kube-api-access-7g6lq,VolumeSource:VolumeSource{HostPath:nil,EmptyDir:nil,GCEPersistentDisk:nil,AWSElasticBlockStore:nil,GitRepo:nil,Secret:nil,NFS:nil,ISCSI:nil,Glusterfs:nil,PersistentVolumeClaim:nil,RBD:nil,FlexVolume:nil,Cinder:nil,CephFS:nil,Flocker:nil,DownwardAPI:nil,FC:nil,AzureFile:nil,ConfigMap:nil,VsphereVolume:nil,Quobyte:nil,AzureDisk:nil,PhotonPersistentDisk:nil,PortworxVolume:nil,ScaleIO:nil,Projected:&ProjectedVolumeSource{Sources:[]VolumeProjection{VolumeProjection{Secret:nil,DownwardAPI:nil,ConfigMap:nil,ServiceAccountToken:&ServiceAccountTokenProjection{Audience:,ExpirationSeconds:*3607,Path:token,},ClusterTrustBundle:nil,},VolumeProjection{Secret:nil,DownwardAPI:nil,ConfigMap:&ConfigMapProjection{LocalObjectReference:LocalObjectReference{Name:kube-root-ca.crt,},Items:[]KeyToPath{KeyToPath{Key:ca.crt,Path:ca.crt,Mode:nil,},},Optional:nil,},ServiceAccountToken:nil,ClusterTrustBundle:nil,},VolumeProjection{Secret:nil,DownwardAPI:&DownwardAPIProjection{Items:[]DownwardAPIVolumeFile{DownwardAPIVolumeFile{Path:namespace,FieldRef:&ObjectFieldSelector{APIVersion:v1,FieldPath:metadata.namespace,},ResourceFieldRef:nil,Mode:nil,},},},ConfigMap:nil,ServiceAccountToken:nil,ClusterTrustBundle:nil,},},DefaultMode:*420,},StorageOS:nil,CSI:nil,Ephemeral:nil,Image:nil,},},},
    Containers:[]Container{
      Container{Name:controller,Image:dev-registry.deckhouse.io/sys/deckhouse-oss@sha256:d1be58b6a9393f9d0e4a894d0eeda7ce2d9da5ad5ab8cd3e0c11f30a9fb9a9a4,Command:[],Args:[/nginx-ingress-controller --configmap=$(POD_NAMESPACE)/nginx-config --v=2 --ingress-class=nginx --healthz-port=10254 --http-port=80 --https-port=443 --update-status=true --publish-service=d8-ingress-nginx/nginx-load-balancer --shutdown-grace-period=120 --validating-webhook=:8443 --validating-webhook-certificate=/etc/nginx/webhook-ssl/tls.crt --validating-webhook-key=/etc/nginx/webhook-ssl/tls.key --controller-class=ingress-nginx.deckhouse.io/nginx --watch-ingress-without-class=true --healthz-host=127.0.0.1 --election-id=ingress-controller-leader-nginx],WorkingDir:,Ports:[]ContainerPort{ContainerPort{Name:,HostPort:0,ContainerPort:80,Protocol:TCP,HostIP:,},ContainerPort{Name:,HostPort:0,ContainerPort:443,Protocol:TCP,HostIP:,},ContainerPort{Name:,HostPort:0,ContainerPort:443,Protocol:UDP,HostIP:,},ContainerPort{Name:webhook,HostPort:0,ContainerPort:8443,Protocol:TCP,HostIP:,},},Env:[]EnvVar{EnvVar{Name:POD_NAME,Value:,ValueFrom:&EnvVarSource{FieldRef:&ObjectFieldSelector{APIVersion:v1,FieldPath:metadata.name,},ResourceFieldRef:nil,ConfigMapKeyRef:nil,SecretKeyRef:nil,},},EnvVar{Name:POD_NAMESPACE,Value:,ValueFrom:&EnvVarSource{FieldRef:&ObjectFieldSelector{APIVersion:v1,FieldPath:metadata.namespace,},ResourceFieldRef:nil,ConfigMapKeyRef:nil,SecretKeyRef:nil,},},EnvVar{Name:POD_IP,Value:127.0.0.1,ValueFrom:nil,},},Resources:ResourceRequirements{Limits:ResourceList{},Requests:ResourceList{cpu: {{23 -3} {<nil>} 23m DecimalSI},ephemeral-storage: {{157286400 0} {<nil>} 150Mi BinarySI},memory: {{209715200 0} {<nil>}  BinarySI},},Claims:[]ResourceClaim{},},VolumeMounts:[]VolumeMount{VolumeMount{Name:client-body-temp-path,ReadOnly:false,MountPath:/chroot/var/lib/nginx/body,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},VolumeMount{Name:fastcgi-temp-path,ReadOnly:false,MountPath:/chroot/var/lib/nginx/fastcgi,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},VolumeMount{Name:proxy-temp-path,ReadOnly:false,MountPath:/chroot/var/lib/nginx/proxy,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},VolumeMount{Name:scgi-temp-path,ReadOnly:false,MountPath:/chroot/var/lib/nginx/scgi,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},VolumeMount{Name:uwsgi-temp-path,ReadOnly:false,MountPath:/chroot/var/lib/nginx/uwsgi,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},VolumeMount{Name:secret-nginx-auth-tls,ReadOnly:false,MountPath:/chroot/etc/nginx/ssl/,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},VolumeMount{Name:tmp-nginx,ReadOnly:false,MountPath:/chroot/tmp/nginx/,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},VolumeMount{Name:webhook-cert,ReadOnly:true,MountPath:/chroot/etc/nginx/webhook-ssl/,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},VolumeMount{Name:kube-api-access-7g6lq,ReadOnly:true,MountPath:/var/run/secrets/kubernetes.io/serviceaccount,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},},LivenessProbe:&Probe{ProbeHandler:ProbeHandler{Exec:nil,HTTPGet:&HTTPGetAction{Path:/controller/healthz,Port:{0 4207 },Host:,Scheme:HTTPS,HTTPHeaders:[]HTTPHeader{},},TCPSocket:nil,GRPC:nil,},InitialDelaySeconds:30,TimeoutSeconds:5,PeriodSeconds:10,SuccessThreshold:1,FailureThreshold:10,TerminationGracePeriodSeconds:nil,},ReadinessProbe:&Probe{ProbeHandler:ProbeHandler{Exec:nil,HTTPGet:&HTTPGetAction{Path:/controller/healthz,Port:{0 4207 },Host:,Scheme:HTTPS,HTTPHeaders:[]HTTPHeader{},},TCPSocket:nil,GRPC:nil,},InitialDelaySeconds:10,TimeoutSeconds:5,PeriodSeconds:2,SuccessThreshold:1,FailureThreshold:3,TerminationGracePeriodSeconds:nil,},Lifecycle:&Lifecycle{PostStart:nil,PreStop:&LifecycleHandler{Exec:&ExecAction{Command:[/wait-shutdown],},HTTPGet:nil,TCPSocket:nil,Sleep:nil,},},TerminationMessagePath:/dev/termination-log,ImagePullPolicy:IfNotPresent,SecurityContext:nil,Stdin:false,StdinOnce:false,TTY:false,EnvFrom:[]EnvFromSource{},TerminationMessagePolicy:File,VolumeDevices:[]VolumeDevice{},StartupProbe:nil,ResizePolicy:[]ContainerResizePolicy{},RestartPolicy:nil,},
      Container{Name:protobuf-exporter,Image:dev-registry.deckhouse.io/sys/deckhouse-oss@sha256:29242ba712e34c3502812348b663a46a481a4f159866f3b44d917ca6520ff23f,Command:[],Args:[],WorkingDir:,Ports:[]ContainerPort{},Env:[]EnvVar{},Resources:ResourceRequirements{Limits:ResourceList{},Requests:ResourceList{cpu: {{11 -3} {<nil>} 11m DecimalSI},ephemeral-storage: {{52428800 0} {<nil>} 50Mi BinarySI},memory: {{23574998 0} {<nil>} 23574998 DecimalSI},},Claims:[]ResourceClaim{},},VolumeMounts:[]VolumeMount{VolumeMount{Name:telemetry-config-file,ReadOnly:false,MountPath:/var/files,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},VolumeMount{Name:kube-api-access-7g6lq,ReadOnly:true,MountPath:/var/run/secrets/kubernetes.io/serviceaccount,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},},LivenessProbe:nil,ReadinessProbe:nil,Lifecycle:nil,TerminationMessagePath:/dev/termination-log,ImagePullPolicy:IfNotPresent,SecurityContext:nil,Stdin:false,StdinOnce:false,TTY:false,EnvFrom:[]EnvFromSource{},TerminationMessagePolicy:File,VolumeDevices:[]VolumeDevice{},StartupProbe:nil,ResizePolicy:[]ContainerResizePolicy{},RestartPolicy:nil,},
      Container{Name:kube-rbac-proxy,Image:dev-registry.deckhouse.io/sys/deckhouse-oss@sha256:e4b34c781c6e299f206ca73c5e62f9967f14295947ed6dc55f0b044eead84581,Command:[],Args:[--secure-listen-address=$(KUBE_RBAC_PROXY_LISTEN_ADDRESS):4207 --v=2 --logtostderr=true --stale-cache-interval=1h30m],WorkingDir:,Ports:[]ContainerPort{ContainerPort{Name:https-metrics,HostPort:0,ContainerPort:4207,Protocol:TCP,HostIP:,},},Env:[]EnvVar{EnvVar{Name:KUBE_RBAC_PROXY_LISTEN_ADDRESS,Value:,ValueFrom:&EnvVarSource{FieldRef:&ObjectFieldSelector{APIVersion:v1,FieldPath:status.podIP,},ResourceFieldRef:nil,ConfigMapKeyRef:nil,SecretKeyRef:nil,},},EnvVar{Name:KUBE_RBAC_PROXY_CONFIG,Value:excludePaths:
- /controller/healthz
upstreams:
- upstream: http://127.0.0.1:10254/
  path: /controller/
  authorization:
    resourceAttributes:
      namespace: d8-ingress-nginx
      apiGroup: apps
      apiVersion: v1
      resource: daemonsets
      subresource: prometheus-controller-metrics
      name: ingress-nginx
- upstream: http://127.0.0.1:9091/metrics
  path: /protobuf/metrics
  authorization:
    resourceAttributes:
      namespace: d8-ingress-nginx
      apiGroup: apps
      apiVersion: v1
      resource: daemonsets
      subresource: prometheus-protobuf-metrics
      name: ingress-nginx
,ValueFrom:nil,},},Resources:ResourceRequirements{Limits:ResourceList{},Requests:ResourceList{cpu: {{11 -3} {<nil>} 11m DecimalSI},ephemeral-storage: {{52428800 0} {<nil>} 50Mi BinarySI},memory: {{23574998 0} {<nil>} 23574998 DecimalSI},},Claims:[]ResourceClaim{},},VolumeMounts:[]VolumeMount{VolumeMount{Name:kube-api-access-7g6lq,ReadOnly:true,MountPath:/var/run/secrets/kubernetes.io/serviceaccount,SubPath:,MountPropagation:nil,SubPathExpr:,RecursiveReadOnly:nil,},},LivenessProbe:nil,ReadinessProbe:nil,Lifecycle:&Lifecycle{PostStart:nil,PreStop:&LifecycleHandler{Exec:&ExecAction{Command:[/controller-probe -server 127.0.0.1:10254],},HTTPGet:nil,TCPSocket:nil,Sleep:nil,},},TerminationMessagePath:/dev/termination-log,ImagePullPolicy:IfNotPresent,SecurityContext:nil,Stdin:false,StdinOnce:false,TTY:false,EnvFrom:[]EnvFromSource{},TerminationMessagePolicy:File,VolumeDevices:[]VolumeDevice{},StartupProbe:nil,ResizePolicy:[]ContainerResizePolicy{},RestartPolicy:nil,},
    },
	EphemeralContainers:,
    RestartPolicy:Always,
    TerminationGracePeriodSeconds:*420,
    ActiveDeadlineSeconds:nil,
    DNSPolicy:ClusterFirst,
    NodeSelector:map[string]string{node.deckhouse.io/group: worker,},
    ServiceAccountName:ingress-nginx,
    DeprecatedServiceAccount:ingress-nginx,
    NodeName:d-shipkov-worker-2,
    HostNetwork:false,
    HostPID:false,
    HostIPC:false,
    SecurityContext:&PodSecurityContext{SELinuxOptions:nil,RunAsUser:nil,RunAsNonRoot:nil,SupplementalGroups:[],FSGroup:nil,RunAsGroup:nil,Sysctls:[]Sysctl{},WindowsOptions:nil,FSGroupChangePolicy:nil,SeccompProfile:nil,AppArmorProfile:nil,SupplementalGroupsPolicy:nil,SELinuxChangePolicy:nil,},
    ImagePullSecrets:[]LocalObjectReference{LocalObjectReference{Name:deckhouse-registry,},},
    Hostname:,
    Subdomain:,
    Affinity:&Affinity{NodeAffinity:&NodeAffinity{RequiredDuringSchedulingIgnoredDuringExecution:&NodeSelector{NodeSelectorTerms:[]NodeSelectorTerm{NodeSelectorTerm{MatchExpressions:[]NodeSelectorRequirement{},MatchFields:[]NodeSelectorRequirement{NodeSelectorRequirement{Key:metadata.name,Operator:In,Values:[d-shipkov-worker-2],},},},},},PreferredDuringSchedulingIgnoredDuringExecution:[]PreferredSchedulingTerm{},},PodAffinity:nil,PodAntiAffinity:nil,},
    SchedulerName:default-scheduler,
    InitContainers:[]Container{},
    AutomountServiceAccountToken:nil,
    Tolerations:[]Toleration{Toleration{Key:dedicated.deckhouse.io,Operator:Equal,Value:ingress-nginx,Effect:,TolerationSeconds:nil,},Toleration{Key:dedicated.deckhouse.io,Operator:Equal,Value:frontend,Effect:,TolerationSeconds:nil,},Toleration{Key:drbd.linbit.com/lost-quorum,Operator:,Value:,Effect:,TolerationSeconds:nil,},Toleration{Key:drbd.linbit.com/force-io-error,Operator:,Value:,Effect:,TolerationSeconds:nil,},Toleration{Key:drbd.linbit.com/ignore-fail-over,Operator:,Value:,Effect:,TolerationSeconds:nil,},Toleration{Key:node.kubernetes.io/not-ready,Operator:Exists,Value:,Effect:NoExecute,TolerationSeconds:nil,},Toleration{Key:node.kubernetes.io/unreachable,Operator:Exists,Value:,Effect:NoExecute,TolerationSeconds:nil,},Toleration{Key:node.kubernetes.io/disk-pressure,Operator:Exists,Value:,Effect:NoSchedule,TolerationSeconds:nil,},Toleration{Key:node.kubernetes.io/memory-pressure,Operator:Exists,Value:,Effect:NoSchedule,TolerationSeconds:nil,},Toleration{Key:node.kubernetes.io/pid-pressure,Operator:Exists,Value:,Effect:NoSchedule,TolerationSeconds:nil,},Toleration{Key:node.kubernetes.io/unschedulable,Operator:Exists,Value:,Effect:NoSchedule,TolerationSeconds:nil,},},
    HostAliases:[]HostAlias{},
    PriorityClassName:system-cluster-critical,
    Priority:*2000000000,
    DNSConfig:nil,
    ShareProcessNamespace:nil,
    ReadinessGates:[]PodReadinessGate{},
    RuntimeClassName:nil,
    EnableServiceLinks:*true,
    PreemptionPolicy:*PreemptLowerPriority,
    Overhead:ResourceList{},
    TopologySpreadConstraints:[]TopologySpreadConstraint{},
    EphemeralContainers:[]EphemeralContainer{},
    SetHostnameAsFQDN:nil,
    OS:nil,
    HostUsers:nil,
    SchedulingGates:[]PodSchedulingGate{},
    ResourceClaims:[]PodResourceClaim{},
    Resources:nil,
  },
  Status:PodStatus{....},
}

"f:metadata":{
  "f:generateName":{},
  "f:labels":{".":{},"f:app":{},"f:controller-revision-hash":{},"f:name":{}},
  "f:ownerReferences":{".":{},"k:{\"uid\":\"ba9f69e0-a104-442a-9362-8aee8d3da964\"}":{}}
},
"f:spec":{
  "f:affinity":{".":{},"f:nodeAffinity":{".":{},"f:requiredDuringSchedulingIgnoredDuringExecution":{}}},
  "f:containers":{
    "k:{\"name\":\"controller\"}":{".":{},"f:args":{},"f:env":{".":{},"k:{\"name\":\"POD_IP\"}":{".":{},"f:name":{},"f:value":{}},"k:{\"name\":\"POD_NAME\"}":{".":{},"f:name":{},"f:valueFrom":{".":{},"f:fieldRef":{}}},"k:{\"name\":\"POD_NAMESPACE\"}":{".":{},"f:name":{},"f:valueFrom":{".":{},"f:fieldRef":{}}}},"f:image":{},"f:imagePullPolicy":{},"f:lifecycle":{".":{},"f:preStop":{".":{},"f:exec":{".":{},"f:command":{}}}},"f:livenessProbe":{".":{},"f:failureThreshold":{},"f:httpGet":{".":{},"f:path":{},"f:port":{},"f:scheme":{}},"f:initialDelaySeconds":{},"f:periodSeconds":{},"f:successThreshold":{},"f:timeoutSeconds":{}},"f:name":{},"f:ports":{".":{},"k:{\"containerPort\":80,\"protocol\":\"TCP\"}":{".":{},"f:containerPort":{},"f:protocol":{}},"k:{\"containerPort\":443,\"protocol\":\"TCP\"}":{".":{},"f:containerPort":{},"f:protocol":{}},"k:{\"containerPort\":443,\"protocol\":\"UDP\"}":{".":{},"f:containerPort":{},"f:protocol":{}},"k:{\"containerPort\":8443,\"protocol\":\"TCP\"}":{".":{},"f:containerPort":{},"f:name":{},"f:protocol":{}}},"f:readinessProbe":{".":{},"f:failureThreshold":{},"f:httpGet":{".":{},"f:path":{},"f:port":{},"f:scheme":{}},"f:initialDelaySeconds":{},"f:periodSeconds":{},"f:successThreshold":{},"f:timeoutSeconds":{}},"f:resources":{".":{},"f:requests":{".":{},"f:ephemeral-storage":{}}},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{},"f:volumeMounts":{".":{},"k:{\"mountPath\":\"/chroot/etc/nginx/ssl/\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/etc/nginx/webhook-ssl/\"}":{".":{},"f:mountPath":{},"f:name":{},"f:readOnly":{}},"k:{\"mountPath\":\"/chroot/tmp/nginx/\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/var/lib/nginx/body\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/var/lib/nginx/fastcgi\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/var/lib/nginx/proxy\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/var/lib/nginx/scgi\"}":{".":{},"f:mountPath":{},"f:name":{}},"k:{\"mountPath\":\"/chroot/var/lib/nginx/uwsgi\"}":{".":{},"f:mountPath":{},"f:name":{}}}},
    "k:{\"name\":\"kube-rbac-proxy\"}":{".":{},"f:args":{},"f:env":{".":{},"k:{\"name\":\"KUBE_RBAC_PROXY_CONFIG\"}":{".":{},"f:name":{},"f:value":{}},"k:{\"name\":\"KUBE_RBAC_PROXY_LISTEN_ADDRESS\"}":{".":{},"f:name":{},"f:valueFrom":{".":{},"f:fieldRef":{}}}},"f:image":{},"f:imagePullPolicy":{},"f:lifecycle":{".":{},"f:preStop":{".":{},"f:exec":{".":{},"f:command":{}}}},"f:name":{},"f:ports":{".":{},"k:{\"containerPort\":4207,\"protocol\":\"TCP\"}":{".":{},"f:containerPort":{},"f:name":{},"f:protocol":{}}},"f:resources":{".":{},"f:requests":{".":{},"f:cpu":{},"f:ephemeral-storage":{},"f:memory":{}}},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{}},
    "k:{\"name\":\"protobuf-exporter\"}":{".":{},"f:image":{},"f:imagePullPolicy":{},"f:name":{},"f:resources":{".":{},"f:requests":{".":{},"f:cpu":{},"f:ephemeral-storage":{},"f:memory":{}}},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{},"f:volumeMounts":{".":{},"k:{\"mountPath\":\"/var/files\"}":{".":{},"f:mountPath":{},"f:name":{}}}}
  },
  "f:dnsPolicy":{},
  "f:enableServiceLinks":{},
  "f:imagePullSecrets":{".":{},"k:{\"name\":\"deckhouse-registry\"}":{}},
  "f:nodeSelector":{},
  "f:priorityClassName":{},
  "f:restartPolicy":{},
  "f:schedulerName":{},
  "f:securityContext":{},
  "f:serviceAccount":{},
  "f:serviceAccountName":{},
  "f:terminationGracePeriodSeconds":{},
  "f:tolerations":{},
  "f:volumes":{".":{},"k:{\"name\":\"client-body-temp-path\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"fastcgi-temp-path\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"proxy-temp-path\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"scgi-temp-path\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"secret-nginx-auth-tls\"}":{".":{},"f:name":{},"f:secret":{".":{},"f:defaultMode":{},"f:secretName":{}}},"k:{\"name\":\"telemetry-config-file\"}":{".":{},"f:configMap":{".":{},"f:defaultMode":{},"f:name":{}},"f:name":{}},"k:{\"name\":\"tmp-nginx\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"uwsgi-temp-path\"}":{".":{},"f:emptyDir":{},"f:name":{}},"k:{\"name\":\"webhook-cert\"}":{".":{},"f:name":{},"f:secret":{".":{},"f:defaultMode":{},"f:secretName":{}}}}
}}B��
*/

/*
	err = pod.Unmarshal([]byte(fmt.Sprintf(`
apiVersion: v1
kind: Pod
metadata:
  annotations:
    cni.cilium.io/ipAddress: 10.10.10.180
    descheduler.alpha.kubernetes.io/request-evict-only: ""
    kubectl.kubernetes.io/default-container: compute
    kubevirt.internal.virtualization.deckhouse.io/allow-pod-bridge-network-live-migration: "true"
    kubevirt.internal.virtualization.deckhouse.io/domain: vm1
    kubevirt.internal.virtualization.deckhouse.io/migrationTransportUnix: "true"
    kubevirt.internal.virtualization.deckhouse.io/vm-generation: "1"
    post.hook.backup.velero.io/command: '["/usr/bin/virt-freezer", "--unfreeze", "--name",
      "vm1", "--namespace", "test1"]'
    post.hook.backup.velero.io/container: compute
    pre.hook.backup.velero.io/command: '["/usr/bin/virt-freezer", "--freeze", "--name",
      "vm1", "--namespace", "test1"]'
    pre.hook.backup.velero.io/container: compute
  creationTimestamp: "2024-12-05T08:53:15Z"
  finalizers:
  - virtualization.deckhouse.io/pod-protection
  generateName: virt-launcher-vm1-
  labels:
    kubevirt.internal.virtualization.deckhouse.io: virt-launcher
    kubevirt.internal.virtualization.deckhouse.io/created-by: c8d3863d-df1b-4414-a6d3-4e5ffa8937ba
    service: v1
    vm: linux
    vm.kubevirt.internal.virtualization.deckhouse.io/name: vm1
  name: virt-launcher-vm1-cjj7d
  namespace: test1
  ownerReferences:
  - apiVersion: internal.virtualization.deckhouse.io/v1
    blockOwnerDeletion: true
    controller: true
    kind: InternalVirtualizationVirtualMachineInstance
    name: vm1
    uid: c8d3863d-df1b-4414-a6d3-4e5ffa8937ba
  resourceVersion: "20673922"
  uid: 9c928f25-5027-43ee-9008-59337b7f2dda
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: node-role.kubernetes.io/control-plane
            operator: DoesNotExist
  automountServiceAccountToken: false
  containers:
  - command:
    - /usr/bin/virt-launcher-monitor
    - --qemu-timeout
    - 342s
    - --name
    - vm1
    - --uid
    - c8d3863d-df1b-4414-a6d3-4e5ffa8937ba
    - --namespace
    - test1
    - --kubevirt-share-dir
    - /var/run/kubevirt
    - --ephemeral-disk-dir
    - /var/run/kubevirt-ephemeral-disks
    - --container-disk-dir
    - /var/run/kubevirt/container-disks
    - --grace-period-seconds
    - "75"
    - --hook-sidecars
    - "0"
    - --ovmf-path
    - /usr/share/OVMF
    env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          apiVersion: v1
          fieldPath: metadata.name
    image: dev-registry.deckhouse.io/sys/deckhouse-oss/modules/virtualization@sha256:909365661b7d011c784d77b5468e7dd156d01fdb9e3eeaf6242912327f85865f
    imagePullPolicy: IfNotPresent
    name: compute
    resources:
      limits:
        cpu: "4"
        devices.virtualization.deckhouse.io/kvm: "1"
        devices.virtualization.deckhouse.io/tun: "1"
        devices.virtualization.deckhouse.io/vhost-net: "1"
        memory: 8574Mi
      requests:
        cpu: "4"
        devices.virtualization.deckhouse.io/kvm: "1"
        devices.virtualization.deckhouse.io/tun: "1"
        devices.virtualization.deckhouse.io/vhost-net: "1"
        ephemeral-storage: 50M
        memory: 8574Mi
    securityContext:
      capabilities:
        add:
        - NET_BIND_SERVICE
        - SYS_NICE
      privileged: false
      runAsNonRoot: false
      runAsUser: 0
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeDevices:
    - devicePath: /dev/vd-vm1-system
      name: vd-vm1-system
    volumeMounts:
    - mountPath: /var/run/kubevirt-private
      name: private
    - mountPath: /var/run/kubevirt
      name: public
    - mountPath: /var/run/kubevirt-ephemeral-disks
      name: ephemeral-disks
    - mountPath: /var/run/kubevirt/container-disks
      mountPropagation: HostToContainer
      name: container-disks
    - mountPath: /var/run/libvirt
      name: libvirt-runtime
    - mountPath: /var/run/kubevirt/sockets
      name: sockets
    - mountPath: /var/run/kubevirt/hotplug-disks
      mountPropagation: HostToContainer
      name: hotplug-disks
  dnsPolicy: ClusterFirst
  enableServiceLinks: false
  hostname: vm1
  nodeSelector:
    cpu-model.node.virtualization.deckhouse.io/Nehalem: "true"
    kubernetes.io/arch: amd64
    kubevirt.internal.virtualization.deckhouse.io/schedulable: "true"
  preemptionPolicy: PreemptLowerPriority
  priority: 1000
  priorityClassName: develop
  readinessGates:
  - conditionType: kubevirt.io/virtual-machine-unpaused
  restartPolicy: Never
  schedulerName: default-scheduler
  securityContext:
    runAsUser: 0
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 90
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  - effect: NoSchedule
    key: node.kubernetes.io/memory-pressure
    operator: Exists
  - effect: NoSchedule
    key: devices.virtualization.deckhouse.io/kvm
    operator: Exists
  - effect: NoSchedule
    key: devices.virtualization.deckhouse.io/tun
    operator: Exists
  - effect: NoSchedule
    key: devices.virtualization.deckhouse.io/vhost-net
    operator: Exists
  volumes:
  - emptyDir: {}
    name: private
  - emptyDir: {}
    name: public
  - emptyDir: {}
    name: sockets
  - emptyDir: {}
    name: virt-bin-share-dir
  - emptyDir: {}
    name: libvirt-runtime
  - emptyDir: {}
    name: ephemeral-disks
  - emptyDir: {}
    name: container-disks
  - name: vd-vm1-system
    persistentVolumeClaim:
      claimName: vd-vm1-system-5ccfc7f3-c98b-492d-9dbf-3555534e9d2a
  - emptyDir: {}
    name: hotplug-disks
	`, nodeSelector)))
	fmt.Printf("ERR: %#v\n", err)
*/

//	err := pod.Unmarshal([]byte(fmt.Sprintf(`
//		{
//          "Name": "hello",
//		  "spec": {
//            "Name": "hello",
//		    "hostPID": true,
//		    "hostNetwork": true,
//		    %s
//		    "tolerations": [{
//		        "operator": "Exists"
//		    }],
//		    "containers": [
//		      {
//		        "name": "nsenter",
//		        "image": "alexeiled/nsenter:2.34",
//		        "command": [
//		          "/nsenter", "--all", "--target=1", "--", "su", "-"
//		        ],
//		        "stdin": true,
//		        "tty": true,
//		        "securityContext": {
//		          "privileged": true
//		        }
//		      }
//		    ]
//		  }
//		}
//	`, nodeSelector)))
//	fmt.Printf("ERR: %#v\n", err)



//	// Using a typed object.
//	pod := &corev1.Pod{
//		ObjectMeta: metav1.ObjectMeta{
//			Namespace: "namespace",
//			Name:      "name",
//		},
//		Spec: corev1.PodSpec{
//			Containers: []corev1.Container{
//				{
//					Image: "nginx",
//					Name:  "nginx",
//				},
//			},
//		},
//	}
//	// c is a created client.
//	_ = c.Create(context.Background(), pod)
//
//	// Using a unstructured object.
//	u := &unstructured.Unstructured{}
//	u.Object = map[string]interface{}{
//		"metadata": map[string]interface{}{
//			"name":      "name",
//			"namespace": "namespace",
//		},
//		"spec": map[string]interface{}{
//			"replicas": 2,
//			"selector": map[string]interface{}{
//				"matchLabels": map[string]interface{}{
//					"foo": "bar",
//				},
//			},
//			"template": map[string]interface{}{
//				"labels": map[string]interface{}{
//					"foo": "bar",
//				},
//				"spec": map[string]interface{}{
//					"containers": []map[string]interface{}{
//						{
//							"name":  "nginx",
//							"image": "nginx",
//						},
//					},
//				},
//			},
//		},
//	}
//	u.SetGroupVersionKind(schema.GroupVersionKind{
//		Group:   "apps",
//		Kind:    "Deployment",
//		Version: "v1",
//	})
//	_ = c.Create(context.Background(), u)
}
