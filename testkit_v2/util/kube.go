package integration

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"testing"

	logr "github.com/go-logr/logr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlrtlog "sigs.k8s.io/controller-runtime/pkg/log"

	appsapi "k8s.io/api/apps/v1"
	coreapi "k8s.io/api/core/v1"
	storapi "k8s.io/api/storage/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	apirtschema "k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"

	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	srv "github.com/deckhouse/sds-replicated-volume/api/v1alpha1"
	virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
)

/*  Config  */

func NewRestConfig(configPath, clusterName string) (*rest.Config, error) {
	var cfg *rest.Config
	var loader clientcmd.ClientConfigLoader
	if configPath != "" {
		loader = &clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath}
	} else {
		loader = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	overrides := clientcmd.ConfigOverrides{}
	if clusterName != "" {
		overrides.Context.Cluster = clusterName
		overrides.CurrentContext = clusterName
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader, &overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed create rest config: %w", err)
	}

	return cfg, nil
}

/*  Kuber Client  */

func NewKubeRTClient(configPath, clusterName string) (ctrlrtclient.Client, error) {
	cfg, err := NewRestConfig(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	// Add options
	var resourcesSchemeFuncs = []func(*apiruntime.Scheme) error{
		virt.AddToScheme,
		srv.AddToScheme,
		snc.AddToScheme,
		kubescheme.AddToScheme,
		extapi.AddToScheme,
		coreapi.AddToScheme,
		storapi.AddToScheme,
		DhSchemeBuilder.AddToScheme,
	}

	scheme := apiruntime.NewScheme()
	for _, f := range resourcesSchemeFuncs {
		err = f(scheme)
		if err != nil {
			return nil, err
		}
	}
	clientOpts := ctrlrtclient.Options{
		Scheme: scheme,
	}

	// Init client
	ctrlrtlog.SetLogger(logr.FromContextOrDiscard(context.Background()))
	cl, err := ctrlrtclient.New(cfg, clientOpts)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

func NewKubeGoClient(configPath, clusterName string) (*kubernetes.Clientset, error) {
	cfg, err := NewRestConfig(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	cl, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

func NewKubeDyClient(configPath, clusterName string) (*dynamic.DynamicClient, error) {
	cfg, err := NewRestConfig(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	cl, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return cl, nil
}

/*  Kuber Cluster object  */

type KCluster struct {
	name       string
	ctx        context.Context
	rtClient   ctrlrtclient.Client
	goClient   *kubernetes.Clientset
	dyClient   *dynamic.DynamicClient
	groupNodes map[string][]string
	nodeGroups map[string][]string
}

func InitKCluster(configPath, clusterName string) (*KCluster, error) {
	if clusterName == "" {
		clusterName = *clusterNameFlag
	}
	if configPath == "" {
		configPath = NestedClusterKubeConfig
	}

	rcl, err := NewKubeRTClient(configPath, clusterName)
	if err != nil {
		Critf("Can't connect cluster %s", clusterName)
		return nil, err
	}

	gcl, err := NewKubeGoClient(configPath, clusterName)
	if err != nil {
		Critf("Can't connect cluster %s", clusterName)
		return nil, err
	}

	dcl, err := NewKubeDyClient(configPath, clusterName)
	if err != nil {
		Critf("Can't connect cluster %s", clusterName)
		return nil, err
	}

	clr := KCluster{
		name:       clusterName,
		ctx:        context.Background(),
		rtClient:   rcl,
		goClient:   gcl,
		dyClient:   dcl,
		groupNodes: make(map[string][]string, len(NodeRequired)),
		nodeGroups: make(map[string][]string),
	}

	return &clr, nil
}

func (clr *KCluster) initGroupNodes(filters map[string]NodeFilter) error {
	for k, f := range filters {
		nodeMap, err := clr.GetNodes(f)
		if err != nil {
			return err
		}

		if len(nodeMap) == 0 {
			Critf("0 Nodes for %s", k)
			if !SkipOptional {
				clr.groupNodes[k] = nil
			}
			continue
		}
		for nodeName := range nodeMap {
			clr.groupNodes[k] = append(clr.groupNodes[k], nodeName)
			clr.nodeGroups[nodeName] = append(clr.nodeGroups[nodeName], k)
		}
		Infof("%d Nodes for %s", len(clr.groupNodes[k]), k)
	}

	return nil
}

func (clr *KCluster) GetGroupNodes(filters ...NodeFilter) map[string][]string {
	if len(clr.groupNodes) == 0 {
		_ = clr.initGroupNodes(NodeRequired)
	}
	if len(filters) == 0 {
		return clr.groupNodes
	}

	resp := map[string][]string{}
	for g, nList := range clr.groupNodes {
		for _, f := range filters {
			if f.NodeGroup.isValid(g) {
				resp[g] = nList
				break
			}
		}
	}

	return resp
}

/*  Node Group  */

var (
	nodeGroupResource = apirtschema.GroupVersionResource{Group: "deckhouse.io", Version: "v1", Resource: "nodegroups"}
)

func (clr *KCluster) GetNodeGroups() ([]unstructured.Unstructured, error) {
	ngs, err := clr.dyClient.Resource(nodeGroupResource).
		List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return ngs.Items, err
}

func (clr *KCluster) CreateNodeGroup(data map[string]interface{}) error {
	name := data["metadata"].(map[string]interface{})["name"]
	obj := unstructured.Unstructured{}
	obj.SetUnstructuredContent(data)

	_, err := clr.dyClient.Resource(nodeGroupResource).
		Create(clr.ctx, &obj, metav1.CreateOptions{})
	if err == nil {
		Infof("NodeGroup %q created", name)
		return nil
	}

	if apierrors.IsAlreadyExists(err) {
		Infof("NodeGroup %q updating ...", name)
		content, err := obj.MarshalJSON()
		if err != nil {
			return err
		}
		_, err = clr.dyClient.Resource(nodeGroupResource).
			Patch(clr.ctx, obj.GetName(), apitypes.MergePatchType, content, metav1.PatchOptions{})
		if err != nil {
			return err
		}
		return nil
	}

	return err
}

func (clr *KCluster) CreateNodeGroupStatic(name, role string, count int) error {
	ngObj := map[string]interface{}{
		"apiVersion": "deckhouse.io/v1",
		"kind":       "NodeGroup",
		"metadata": map[string]interface{}{
			"name": name,
		},
		"spec": map[string]interface{}{
			"nodeType": "Static",
			"staticInstances": map[string]interface{}{
				"count": count,
				"labelSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"node-role": role,
					},
				},
			},
		},
	}
	return clr.CreateNodeGroup(ngObj)
}

func (clr *KCluster) DeleteNodeGroup(name string) error {
	err := clr.dyClient.Resource(nodeGroupResource).Delete(clr.ctx, name, metav1.DeleteOptions{})
	if err == nil || apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

/*  Name Space  */

func (clr *KCluster) GetNs(filters ...NsFilter) ([]coreapi.Namespace, error) {
	objs := coreapi.NamespaceList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})
	err := clr.rtClient.List(clr.ctx, &objs, opts)
	if err != nil {
		Errf("Can't get NSs: %s", err.Error())
		return nil, err
	}

	if len(filters) == 0 {
		return objs.Items, nil
	}

	resp := make([]coreapi.Namespace, 0, len(objs.Items))
	for _, ns := range objs.Items {
		for _, f := range filters {
			if !f.Check(ns) {
				continue
			}

			resp = append(resp, ns)
			break
		}
	}

	return resp, nil
}

func (clr *KCluster) CreateNs(nsName string) error {
	namespace := &coreapi.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}

	if err := clr.rtClient.Create(clr.ctx, namespace); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}

		Errf("Can't create NS %s", nsName)
		return err
	}
	return nil
}

func (clr *KCluster) DeleteNs(nsName string) error {
	namespace := coreapi.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}
	return clr.rtClient.Delete(clr.ctx, &namespace)
}

/*  Node  */

func (clr *KCluster) GetNodes(filters ...NodeFilter) (map[string]coreapi.Node, error) {
	resp := make(map[string]coreapi.Node)

	nodes, err := (*clr.goClient).CoreV1().Nodes().List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		Errf("Can't get Nodes: %s", err.Error())
		return nil, err
	}

	for _, node := range nodes.Items {
		if len(filters) == 0 {
			resp[node.ObjectMeta.Name] = node
			continue
		}

		for _, f := range filters {
			if !f.Check(node) {
				continue
			}

			ng := clr.nodeGroups[node.ObjectMeta.Name]
			if !f.Intersec(ng) {
				continue
			}

			resp[node.ObjectMeta.Name] = node
			break
		}
	}

	return resp, nil
}

/*  Block Device  */

func (clr *KCluster) GetBDs(filters ...BdFilter) (map[string]snc.BlockDevice, error) {
	resp := make(map[string]snc.BlockDevice)

	bdList := &snc.BlockDeviceList{}
	err := clr.rtClient.List(clr.ctx, bdList)
	if err != nil {
		Errf("Can't get BDs: %s", err.Error())
		return nil, err
	}

	for _, bd := range bdList.Items {
		if len(filters) == 0 {
			resp[bd.Name] = bd
			continue
		}

		for _, f := range filters {
			if f.Check(bd) {
				resp[bd.Name] = bd
				break
			}
		}
	}

	return resp, nil
}

/* Virtual Machines */

type VM struct {
	Name   string
	Status virt.MachinePhase
}

func (clr *KCluster) GetVMs(nsName string) ([]VM, error) {
	objs := virt.VirtualMachineList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})
	err := clr.rtClient.List(clr.ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	vmList := []VM{}
	for _, item := range objs.Items {
		vmList = append(vmList, VM{Name: item.Name, Status: item.Status.Phase})
	}

	return vmList, nil
}

func (clr *KCluster) CreateVM(
	nsName string,
	vmName string,
	ip string,
	cpu int,
	memory string,
	storageClass string,
	imgUrl string,
	sshPubKey string,
	systemDriveSize int64,
) error {
	splittedUrl := strings.Split(imgUrl, "/")
	CVMIName := strings.Split(splittedUrl[len(splittedUrl)-1], ".")[0]
	vmCVMI := &virt.ClusterVirtualImage{}
	CVMIList, err := clr.GetCVMIs(CVMIName)
	if err != nil {
		return err
	}
	if len(CVMIList) == 0 {
		vmCVMI, err = clr.CreateCVMI(CVMIName, imgUrl)
		if err != nil {
			return err
		}
	} else {
		vmCVMI.Name = CVMIList[0].Name
	}

	vmIPClaimName := ""
	if ip != "" {
		vmIPClaim := &virt.VirtualMachineIPAddress{}
		vmIPClaimName = fmt.Sprintf("%s-ipaddress-0", vmName)
		vmIPClaimList, err := clr.ListIPClaim(nsName, vmIPClaimName)
		if err != nil {
			return err
		}
		if len(vmIPClaimList) == 0 {
			vmIPClaim, err = clr.CreateVMIPClaim(nsName, vmIPClaimName, ip)
			if err != nil {
				return err
			}
		} else {
			vmIPClaim = &vmIPClaimList[0]
		}
		vmIPClaimName = vmIPClaim.Name
	}

	vmSystemDisk := &virt.VirtualDisk{}
	vmdName := fmt.Sprintf("%s-system", vmName)
	vmdList, err := clr.GetVMDs(nsName, vmdName)
	if err != nil {
		return err
	}
	if len(vmdList) == 0 {
		vmSystemDisk, err = clr.CreateVMDFromCVMI(nsName, vmdName, storageClass, systemDriveSize, vmCVMI)
		if err != nil {
			return err
		}
	}

	currentMemory, err := resource.ParseQuantity(memory)
	if err != nil {
		return err
	}

	vmObj := &virt.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: nsName,
			Labels:    map[string]string{"vm": "linux", "service": "v1"},
		},
		Spec: virt.VirtualMachineSpec{
			VirtualMachineClassName:  "generic",
			EnableParavirtualization: true,
			RunPolicy:                virt.RunPolicy("AlwaysOn"),
			OsType:                   virt.OsType("Generic"),
			Bootloader:               virt.BootloaderType("BIOS"),
			VirtualMachineIPAddress:  vmIPClaimName,
			CPU:                      virt.CPUSpec{Cores: cpu, CoreFraction: "100%"},
			Memory:                   virt.MemorySpec{Size: currentMemory},
			BlockDeviceRefs: []virt.BlockDeviceSpecRef{
				{
					Kind: virt.DiskDevice,
					Name: vmSystemDisk.Name,
				},
			},
			Provisioning: &virt.Provisioning{
				Type: virt.ProvisioningType("UserData"),
				UserData: fmt.Sprintf(`#cloud-config
package_update: true
packages:
- qemu-guest-agent
runcmd:
- [ hostnamectl, set-hostname, %s ]
- [ systemctl, daemon-reload ]
- [ systemctl, enable, --now, qemu-guest-agent.service ]
user: user
password: user
ssh_pwauth: True
chpasswd: { expire: False }
sudo: ALL=(ALL) NOPASSWD:ALL
chpasswd: { expire: False }
ssh_authorized_keys:
  - %s
`, vmName, sshPubKey),
			},
		},
	}

	err = clr.rtClient.Create(clr.ctx, vmObj)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (clr *KCluster) GetVMBDs(nsName, vmName, vmdName string) ([]virt.VirtualMachineBlockDeviceAttachment, error) {
	vmbdas := virt.VirtualMachineBlockDeviceAttachmentList{}
	optsList := ctrlrtclient.ListOptions{}
	if nsName != "" {
		optsList.Namespace = nsName
	}
	opts := ctrlrtclient.ListOption(&optsList)
	if err := clr.rtClient.List(clr.ctx, &vmbdas, opts); err != nil {
		return nil, err
	}

	if vmName == "" && vmdName == "" {
		return vmbdas.Items, nil
	}

	resp := []virt.VirtualMachineBlockDeviceAttachment{}
	for _, vmbd := range vmbdas.Items {
		if vmName != "" && vmbd.Spec.VirtualMachineName != vmName {
			continue
		}
		if vmdName != "" && vmbd.Name != vmdName {
			continue
		}
		resp = append(resp, vmbd)
	}

	return resp, nil
}

func (clr *KCluster) CheckVMBDs(nsName, vmName, vmdName string) error {
	for i := 0; ; i++ {
		vmbds, err := clr.GetVMBDs(nsName, vmName, vmdName)
		if err != nil {
			return err
		}

		allOk := true
		for _, vmbd := range vmbds {
			if vmbd.Status.Phase != "Attached" {
				allOk = false
				break
			}
		}
		if allOk {
			break
		}

		if i >= retries {
			Fatalf("Timeout waiting VMBD attached")
		}

		Debugf("VMBD %s not Attached", vmbds[0].Name)
		//Debugf("VMBD %s: %#v", vmbd.Name, vmbd.Status)
		time.Sleep(10 * time.Second)
	}
	return nil
}

func (clr *KCluster) AttachVMBD(vmName, vmdName, storageClass string, size int64) error {
	vmdList, err := clr.GetVMDs(TestNS, vmdName)
	if err != nil {
		return err
	}
	if len(vmdList) == 0 {
		_, err = clr.CreateVMD(TestNS, vmdName, storageClass, size)
		if err != nil {
			return err
		}
	}

	err = clr.rtClient.Create(clr.ctx, &virt.VirtualMachineBlockDeviceAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmdName,
			Namespace: TestNS,
		},
		Spec: virt.VirtualMachineBlockDeviceAttachmentSpec{
			VirtualMachineName: vmName,
			BlockDeviceRef: virt.VMBDAObjectRef{
				Kind: "VirtualDisk",
				Name: vmdName,
			},
		},
	})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

/*  SSH Credentials  */

func (clr *KCluster) GetSSHCredentials() ([]SSHCredentials, error) {
	credentials := SSHCredentialsList{}
	err := clr.rtClient.List(clr.ctx, &credentials)
	if err != nil {
		return nil, err
	}

	return credentials.Items, nil
}

func (clr *KCluster) CreateSSHCredentials(name, user, privSshKey string) error {
	sshcredentials := &SSHCredentials{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: SSHCredentialsSpec{
			User:          user,
			PrivateSSHKey: privSshKey,
			//SSHPort: 22,
		},
	}

	if err := clr.rtClient.Create(clr.ctx, sshcredentials); err != nil {
		Errf("Can't create SSHCredentials %s: %s", name, err.Error())
		return err
	}
	return nil
}

func (clr *KCluster) CreateOrUpdSSHCredentials(name, user, privSshKey string) error {
	sshcredentials := &SSHCredentials{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: SSHCredentialsSpec{
			User:          user,
			PrivateSSHKey: privSshKey,
			//SSHPort: 22,
		},
	}

	err := clr.rtClient.Create(clr.ctx, sshcredentials)
	if err == nil {
		return nil
	}
	if !apierrors.IsAlreadyExists(err) {
		Errf("Can't create SSHCredentials %s: %s", name, err.Error())
		return err
	}

	err = clr.rtClient.Get(clr.ctx, ctrlrtclient.ObjectKey{Name: name}, sshcredentials)
	if err != nil {
		Errf("Can't get SSHCredentials %s: %s", name, err.Error())
		return err
	}
	sshcredentials.Spec.User = user
	sshcredentials.Spec.PrivateSSHKey = privSshKey
	if err = clr.rtClient.Update(clr.ctx, sshcredentials); err != nil {
		Errf("Can't update SSHCredentials %s: %s", name, err.Error())
		return err
	}
	return nil
}

func (clr *KCluster) DeleteSSHCredentials(name string) error {
	sshcredentials := &SSHCredentials{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return clr.rtClient.Delete(clr.ctx, sshcredentials)
}

/*  Static Instance  */

func (clr *KCluster) GetStaticInstances() ([]StaticInstance, error) {
	sis := StaticInstanceList{}
	err := clr.rtClient.List(clr.ctx, &sis)
	if err != nil {
		return nil, err
	}

	return sis.Items, nil
}

func (clr *KCluster) CreateStaticInstance(name, role, ip, credentials string) error {
	si := &StaticInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"node-role": role},
		},
		Spec: StaticInstanceSpec{
			Address:        ip,
			CredentialsRef: GetSSHCredentialsRef(credentials),
		},
	}

	if err := clr.rtClient.Create(clr.ctx, si); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}

		Errf("Can't create StaticInstance %s", name)
		return err
	}
	return nil
}

func (clr *KCluster) CreateOrUpdStaticInstance(name, role, ip, credentials string) error {
	Errf("Not implemented function")
	return fmt.Errorf("Not implemented")
}

func (clr *KCluster) DeleteStaticInstance(name string) error {
	si := &StaticInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return clr.rtClient.Delete(clr.ctx, si)
}

/*  Static Node  */

func (clr *KCluster) AddStaticNodes(name, user string, ips []string) error {
	privSshKey, err := os.ReadFile(filepath.Join(KubePath, PrivKeyName))
	if err != nil {
		Errf("Read %s: %s", PrivKeyName, err.Error())
		return err
	}
	b64SshKey := base64.StdEncoding.EncodeToString(privSshKey)

	credentialName := name + "rsa"
	if err = clr.CreateOrUpdSSHCredentials(credentialName, user, b64SshKey); err != nil {
		Errf("Create SSHCredential: %s", err.Error())
		return err
	}

	role := fmt.Sprintf("vm-%s-worker", name)
	for _, ip := range ips {
		siName := fmt.Sprintf("si-%s-%s", name, hashMd5(ip)[:8])
		err = clr.CreateStaticInstance(siName, role, ip, credentialName)
		if err != nil {
			Errf("Create StaticInstance %s: %s", siName, err.Error())
			return err
		}
	}

	if err = clr.CreateNodeGroupStatic(role, role, len(ips)); err != nil {
		Errf("Create NodeGroup: %s", err.Error())
		return err
	}

	return nil
}

/*  CVMI  */

type CVMI struct {
	Name string
}

func (clr *KCluster) GetCVMIs(CVMISearch string) ([]CVMI, error) {
	objs := virt.ClusterVirtualImageList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})
	if err := clr.rtClient.List(clr.ctx, &objs, opts); err != nil {
		return nil, err
	}

	cvmiList := []CVMI{}
	for _, item := range objs.Items {
		if CVMISearch == "" || CVMISearch == item.Name {
			cvmiList = append(cvmiList, CVMI{Name: item.Name})
		}
	}

	return cvmiList, nil
}

func (clr *KCluster) ListIPClaim(nsName string, vmIPClaimSearch string) ([]virt.VirtualMachineIPAddress, error) {
	objs := virt.VirtualMachineIPAddressList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})
	err := clr.rtClient.List(clr.ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	vmIPClaimList := []virt.VirtualMachineIPAddress{}
	for _, item := range objs.Items {
		if vmIPClaimSearch == "" || vmIPClaimSearch == item.Name {
			vmIPClaimList = append(vmIPClaimList, item)
		}
	}

	return vmIPClaimList, nil
}

func (clr *KCluster) CreateCVMI(name string, url string) (*virt.ClusterVirtualImage, error) {
	vmCVMI := &virt.ClusterVirtualImage{ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: virt.ClusterVirtualImageSpec{
			DataSource: virt.ClusterVirtualImageDataSource{Type: "HTTP", HTTP: &virt.DataSourceHTTP{URL: url}},
		},
	}

	err := clr.rtClient.Create(clr.ctx, vmCVMI)
	if err != nil {
		return nil, err
	}

	return vmCVMI, nil
}

func (clr *KCluster) CreateVMIPClaim(nsName string, name string, ip string) (*virt.VirtualMachineIPAddress, error) {
	vmClaim := &virt.VirtualMachineIPAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nsName,
		},
		Spec: virt.VirtualMachineIPAddressSpec{
			Type:     virt.VirtualMachineIPAddressTypeStatic,
			StaticIP: ip,
		},
	}

	err := clr.rtClient.Create(clr.ctx, vmClaim)
	if err != nil {
		return nil, err
	}

	return vmClaim, nil
}

/* VMD */

func (clr *KCluster) GetVMDs(nsName string, VMDSearch string) ([]virt.VirtualDisk, error) {
	objs := virt.VirtualDiskList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})

	err := clr.rtClient.List(clr.ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	vmdList := []virt.VirtualDisk{}
	for _, item := range objs.Items {
		if VMDSearch == "" || VMDSearch == item.Name {
			vmdList = append(vmdList, item)
		}
	}

	return vmdList, nil
}

func (clr *KCluster) CreateVMD(nsName string, name string, storageClass string, sizeInGi int64) (*virt.VirtualDisk, error) {
	vmDisk := &virt.VirtualDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nsName,
		},
		Spec: virt.VirtualDiskSpec{
			PersistentVolumeClaim: virt.VirtualDiskPersistentVolumeClaim{
				Size:         resource.NewQuantity(sizeInGi*1024*1024*1024, resource.BinarySI),
				StorageClass: &storageClass,
			},
		},
	}

	err := clr.rtClient.Create(clr.ctx, vmDisk)
	if err != nil {
		return nil, err
	}

	return vmDisk, nil
}

func (clr *KCluster) CreateVMDFromCVMI(nsName string, name string, storageClass string, sizeInGi int64, vmCVMI *virt.ClusterVirtualImage) (*virt.VirtualDisk, error) {
	vmDisk := &virt.VirtualDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nsName,
		},
		Spec: virt.VirtualDiskSpec{
			PersistentVolumeClaim: virt.VirtualDiskPersistentVolumeClaim{
				Size:         resource.NewQuantity(sizeInGi*1024*1024*1024, resource.BinarySI),
				StorageClass: &storageClass,
			},
			DataSource: &virt.VirtualDiskDataSource{
				Type: virt.DataSourceTypeObjectRef,
				ObjectRef: &virt.VirtualDiskObjectRef{
					Kind: virt.ClusterVirtualImageKind,
					Name: vmCVMI.Name,
				},
			},
		},
	}

	err := clr.rtClient.Create(clr.ctx, vmDisk)
	if err != nil {
		return nil, err
	}

	return vmDisk, nil
}

/*  LVM Volume Group  */

func (clr *KCluster) GetLVGs(filters ...LvgFilter) (map[string]snc.LVMVolumeGroup, error) {
	resp := make(map[string]snc.LVMVolumeGroup)

	lvgList := &snc.LVMVolumeGroupList{}
	if err := clr.rtClient.List(clr.ctx, lvgList); err != nil {
		Errf("Can't get LVGs: %s", err.Error())
		return nil, err
	}

	for _, lvg := range lvgList.Items {
		if len(filters) == 0 {
			resp[lvg.Name] = lvg
			continue
		}

		for _, f := range filters {
			if !f.Check(lvg) {
				continue
			}

			resp[lvg.Name] = lvg
			break
		}
	}

	return resp, nil
}

func (clr *KCluster) CreateLVG(name, nodeName, bdName string) (*snc.LVMVolumeGroup, error) {
	lvmVolumeGroup := &snc.LVMVolumeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: snc.LVMVolumeGroupSpec{
			ActualVGNameOnTheNode: name,
			BlockDeviceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "kubernetes.io/metadata.name", Operator: metav1.LabelSelectorOpIn, Values: []string{bdName}},
				},
			},
			Type:  "Local",
			Local: snc.LVMVolumeGroupLocalSpec{NodeName: nodeName},
		},
	}
	err := clr.rtClient.Create(clr.ctx, lvmVolumeGroup)
	if err != nil {
		Errf("Can't create LVG %s (node %s, bd %s)", name, nodeName, bdName)
		return nil, err
	}
	return lvmVolumeGroup, nil
}

func (clr *KCluster) UpdateLVG(lvg *snc.LVMVolumeGroup) error {
	err := clr.rtClient.Update(clr.ctx, lvg)
	if err != nil {
		Errf("Can't update LVG %s", lvg.Name)
		return err
	}

	return nil
}

func (clr *KCluster) DeleteLVG(filters ...LvgFilter) error {
	lvgMap, _ := clr.GetLVGs(filters...)

	for _, lvg := range lvgMap {
		if err := clr.rtClient.Delete(clr.ctx, &lvg); err != nil {
			return err
		}
		Infof("LVG deleted: %s", lvg.Name)
	}

	return nil
}

/*  Storage Class  */

func (clr *KCluster) CreateSC(name string) (*storapi.StorageClass, error) {
	lvmType := "Thick"
	lvmVolGroups := "- name: vg-w1\n- name: vg-w2"

	volBindingMode := storapi.VolumeBindingImmediate

	reclaimPolicy := coreapi.PersistentVolumeReclaimDelete

	volExpansion := true

	sc := &storapi.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Provisioner: "lvm.csi.storage.deckhouse.io",
		Parameters: map[string]string{
			"lvm.csi.storage.deckhouse.io/lvm-type":            lvmType,
			"lvm.csi.storage.deckhouse.io/volume-binding-mode": string(volBindingMode),
			"lvm.csi.storage.deckhouse.io/lvm-volume-groups":   lvmVolGroups,
		},
		ReclaimPolicy:        &reclaimPolicy,
		MountOptions:         nil,
		AllowVolumeExpansion: &volExpansion,
		VolumeBindingMode:    &volBindingMode,
	}

	if err := clr.rtClient.Create(clr.ctx, sc); err != nil {
		Errf("Can't create SC %s", sc.Name)
		return nil, err
	}
	return sc, nil
}

func (clr *KCluster) DeleteSC(name string) error {
	Errf("Not implemented function")
	return nil
}

/*  Daemon Set  */

func (clr *KCluster) GetDaemonSet(nsName, dsName string) (*appsapi.DaemonSet, error) {
	ds, err := (*clr.goClient).AppsV1().DaemonSets(nsName).Get(clr.ctx, dsName, metav1.GetOptions{})
	if err != nil {
		Errf("Can't get '%s.%s' DS: %s", nsName, dsName, err.Error())
		return nil, err
	}

	return ds, nil
}

func (clr *KCluster) GetDaemonSets(nsName string) ([]appsapi.DaemonSet, error) {
	dsList, err := (*clr.goClient).AppsV1().DaemonSets(nsName).List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		Errf("Can't get '%s' DSs: %s", nsName, err.Error())
		return nil, err
	}

	return dsList.Items, nil
}

/*  Persistent Volume Claims  */

func (clr *KCluster) GetPVC(nsName string) ([]coreapi.PersistentVolumeClaim, error) {
	pvcList, err := (*clr.goClient).CoreV1().PersistentVolumeClaims(nsName).List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		Errf("Can't get '%s' PVCs: %s", nsName, err.Error())
		return nil, err
	}

	return pvcList.Items, nil
}

func (clr *KCluster) CreatePVC(name, scName, size string) (*coreapi.PersistentVolumeClaim, error) {
	resourceList := make(map[coreapi.ResourceName]resource.Quantity)
	sizeStorage, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, err
	}
	resourceList[coreapi.ResourceStorage] = sizeStorage
	volMode := coreapi.PersistentVolumeFilesystem

	pvc := coreapi.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       PVCKind,
			APIVersion: PVCAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: TestNS,
		},
		Spec: coreapi.PersistentVolumeClaimSpec{
			StorageClassName: &scName,
			AccessModes: []coreapi.PersistentVolumeAccessMode{
				coreapi.ReadWriteOnce,
			},
			Resources: coreapi.VolumeResourceRequirements{
				Requests: resourceList,
			},
			VolumeMode: &volMode,
		},
	}

	err = clr.rtClient.Create(clr.ctx, &pvc)
	if err != nil {
		return nil, err
	}
	return &pvc, nil
}

func (clr *KCluster) WaitPVCStatus(name string) (string, error) {
	pvc := coreapi.PersistentVolumeClaim{}
	for i := 0; i < PVCWaitIterationCount; i++ {
		err := clr.rtClient.Get(clr.ctx, ctrlrtclient.ObjectKey{
			Name:      name,
			Namespace: TestNS,
		}, &pvc)
		if err != nil {
			Debugf("Get PVC error: %v", err)
		}
		if pvc.Status.Phase == coreapi.ClaimBound {
			return string(pvc.Status.Phase), nil
		}

		if len(pvc.Status.Phase) == 0 {
			return PVCDeletedStatus, nil
		}

		time.Sleep(PVCWaitInterval * time.Second)
	}
	return string(pvc.Status.Phase), fmt.Errorf("the waiting time %d or the pvc to be ready has expired",
		PVCWaitInterval*PVCWaitIterationCount)
}

func (clr *KCluster) DeletePVC(name string) error {
	pvc := coreapi.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       PVCKind,
			APIVersion: PVCAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: TestNS,
		},
	}

	if err := clr.rtClient.Delete(clr.ctx, &pvc); err != nil {
		return err
	}
	return nil
}

func (clr *KCluster) DeletePVCWait(name string) (string, error) {
	Errf("Not implemented function")
	return "", nil
}

func (clr *KCluster) UpdatePVC(pvc *coreapi.PersistentVolumeClaim) error {
	err := clr.rtClient.Update(clr.ctx, pvc)
	if err != nil {
		Errf("Can't update PVC %s", pvc.Name)
		return err
	}

	return nil
}

/*  Virtual Disk  */

func (clr *KCluster) GetVMD(nsName, vmdName string) ([]virt.VirtualDisk, error) {
	vmds := virt.VirtualDiskList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})

	err := clr.rtClient.List(clr.ctx, &vmds, opts)
	if err != nil {
		Errf("Can't get '%s' VMD %s: %s", nsName, vmdName, err.Error())
		return nil, err
	}

	if vmdName == "" {
		return vmds.Items, nil
	}

	vmdList := []virt.VirtualDisk{}
	for _, vmd := range vmds.Items {
		if vmdName == vmd.Name {
			vmdList = append(vmdList, vmd)
		}
	}

	return vmdList, nil
}

func (clr *KCluster) GetTestVMD() ([]virt.VirtualDisk, error) {
	return clr.GetVMD(TestNS, "")
}

func (clr *KCluster) UpdateVMD(vmd *virt.VirtualDisk) error {
	return clr.rtClient.Update(clr.ctx, vmd)
}

/*  Test  */

func (clr *KCluster) Test(t *testing.T) tRunner {
	return tRunner{clr, t}
}
