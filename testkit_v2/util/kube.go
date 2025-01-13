package integration

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"

	// Options
	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	srv "github.com/deckhouse/sds-replicated-volume/api/v1alpha1"
	virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
	coreapi "k8s.io/api/core/v1"
	storapi "k8s.io/api/storage/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
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
	// Override the cluster name if provided.
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

func NewKubeRTClient(configPath, clusterName string) (*ctrlrtclient.Client, error) {
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
	cl, err := ctrlrtclient.New(cfg, clientOpts)
	if err != nil {
		return nil, err
	}

	return &cl, nil
}

func NewKubeGoClient(configPath, clusterName string) (*kubernetes.Clientset, error) {
	// to create the clientset
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

/*  Kuber Cluster object  */

type KCluster struct {
	name       string
	ctx        context.Context
	rtClient   *ctrlrtclient.Client
	goClient   *kubernetes.Clientset
	groupNodes map[string][]string
	nodeGroups map[string][]string
}

func InitKCluster(configPath, clusterName string) (*KCluster, error) {
	configPath = envConfigPath(configPath)
	clusterName = envClusterName(clusterName)

	rcl, err := NewKubeRTClient(configPath, clusterName)
	if err != nil {
		Critf("Can`t connect cluster %s", clusterName)
		return nil, err
	}

	gcl, err := NewKubeGoClient(configPath, clusterName)
	if err != nil {
		Critf("Can`t connect cluster %s", clusterName)
		return nil, err
	}

	clr := KCluster{
		name:       clusterName,
		ctx:        context.Background(),
		rtClient:   rcl,
		goClient:   gcl,
		groupNodes: make(map[string][]string, len(NodeRequired)),
		nodeGroups: make(map[string][]string),
	}

	for k, f := range NodeRequired {
		nodeMap, err := clr.GetNodes(f)
		if err != nil {
			return nil, err
		}

		if len(nodeMap) == 0 {
			Critf("0 Nodes for %s", k)
			// TODO panic for CI/Stage/not debug mode
			clr.groupNodes[k] = nil
			continue
		}
		for nodeName := range nodeMap {
			clr.groupNodes[k] = append(clr.groupNodes[k], nodeName)
			clr.nodeGroups[nodeName] = append(clr.nodeGroups[nodeName], k)
		}
		Infof("%d Nodes for %s", len(clr.groupNodes[k]), k)
	}

	// TODO create test NameSpace

	return &clr, nil
}

func (clr *KCluster) GetGroupNodes(filters ...Filter) map[string][]string {
	if len(filters) == 0 {
		return clr.groupNodes
	}

	resp, f := map[string][]string{}, filters[0]
	fGroup, fNotGroup := f.NodeGroup, f.NotNodeGroup
	for g := range clr.groupNodes {
		if !f.match(g, fGroup, fNotGroup) {
			continue
		}
		f.NodeGroup = []string{g}
		nodeMap, err := clr.GetNodes(f)
		if err != nil {
			return nil
		}
		resp[g] = make([]string, 0, len(nodeMap))
		for k := range nodeMap {
			resp[g] = append(resp[g], k)
		}
	}
	return resp
}

func (clr *KCluster) GetNodeGroups(filter *Filter) map[string][]string {
	return clr.nodeGroups
}

/*  Name Space  */

func (clr *KCluster) GetNs(filters ...Filter) ([]coreapi.Namespace, error) {
	objs := coreapi.NamespaceList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})
	err := (*clr.rtClient).List(clr.ctx, &objs, opts)
	if err != nil {
		Errf("Can`t get NS list")
		return nil, err
	}

	if len(filters) == 0 {
		return objs.Items, nil
	}

	resp, f := make([]coreapi.Namespace, 0, len(objs.Items)), filters[0]
	for _, item := range objs.Items {
		if !f.match(item.Name, f.NS, f.NotNS) {
			continue
		}
		resp = append(resp, item)
	}

	return resp, nil
}

func (clr *KCluster) CreateNs(nsName string) error {
	namespace := &coreapi.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
	}

	if err := (*clr.rtClient).Create(clr.ctx, namespace); err != nil {
		Errf("Can`t create NS %s", nsName)
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
	return (*clr.rtClient).Delete(clr.ctx, &namespace)
}

/*  Node  */

func (clr *KCluster) GetNodes(filters ...Filter) (map[string]coreapi.Node, error) {
	resp := make(map[string]coreapi.Node)

	nodes, err := (*clr.goClient).CoreV1().Nodes().List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		Errf("Can`t get Nodes (%s): %v", clr.name, err)
		return nil, err
	}

	for _, node := range nodes.Items {
		if len(filters) != 0 {
			f := filters[0]
			if !f.checkNode(node) {
				continue
			}

			ng := clr.nodeGroups[node.ObjectMeta.Name]
			if !f.intersec(ng, f.NodeGroup, f.NotNodeGroup) {
				continue
			}
		}

		resp[node.ObjectMeta.Name] = node
	}

	return resp, nil
}

/*  Block Device  */

func (clr *KCluster) GetBDs(filter *Filter) (map[string]snc.BlockDevice, error) {
	resp := make(map[string]snc.BlockDevice)

	bdList := &snc.BlockDeviceList{}
	err := (*clr.rtClient).List(clr.ctx, bdList)
	if err != nil {
		Errf("Can`t get BDs (%s)", clr.name)
		return nil, err
	}

	for _, bd := range bdList.Items {
		if filter == nil {
			resp[bd.Name] = bd
			continue
		}
		if !filter.checkConsumable(bd) {
			continue
		}
		if !filter.checkName(bd.Name) {
			continue
		}
		if !filter.match(bd.Status.NodeName, filter.Node, filter.NotNode) {
			continue
		}
		resp[bd.Name] = bd
	}

	return resp, nil
}

/*  LVM Volume Group  */

func (clr *KCluster) GetLVGs(filter *Filter) (map[string]snc.LVMVolumeGroup, error) {
	resp := make(map[string]snc.LVMVolumeGroup)

	lvgList := &snc.LVMVolumeGroupList{}
	if err := (*clr.rtClient).List(clr.ctx, lvgList); err != nil {
		Errf("Can`t get LVGs (%s)", clr.name)
		return nil, err
	}

	for _, lvg := range lvgList.Items {
		if filter != nil && !filter.like(lvg.Name, filter.Name, filter.NotName) {
			continue
		}
		if len(lvg.Status.Nodes) == 0 {
			continue
		}
		if filter != nil && !filter.match(lvg.Status.Nodes[0].Name, filter.Node, filter.NotNode) {
			continue
		}

		resp[lvg.Name] = lvg
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
	err := (*clr.rtClient).Create(clr.ctx, lvmVolumeGroup)
	if err != nil {
		Errf("Can`t create LVG %s (node %s, bd %s)", name, nodeName, bdName)
		return nil, err
	}
	return lvmVolumeGroup, nil
}

func (clr *KCluster) UpdateLVG(lvg *snc.LVMVolumeGroup) error {
	err := (*clr.rtClient).Update(clr.ctx, lvg)
	if err != nil {
		Errf("Can`t update LVG %s", lvg.Name)
		return err
	}

	return nil
}

func (clr *KCluster) DeleteLVG(filter *Filter) error {
	lvgMap, _ := clr.GetLVGs(filter)

	for _, lvg := range lvgMap {
		if err := (*clr.rtClient).Delete(clr.ctx, &lvg); err != nil {
			return err
		}
		Infof("LVG deleted: %s", lvg.Name)
	}

	return nil
}

func (clr *KCluster) DelTestLVG() error {
	return clr.DeleteLVG(&Filter{Name: []string{"e2e-lvg-"}})
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

	if err := (*clr.rtClient).Create(clr.ctx, sc); err != nil {
		Errf("Can`t create SC %s", sc.Name)
		return nil, err
	}
	return sc, nil
}

/*  Persistent Volume Claims  */

func (clr *KCluster) GetPVC(nsName string) ([]coreapi.PersistentVolumeClaim, error) {
	pvcList, err := (*clr.goClient).CoreV1().PersistentVolumeClaims(nsName).List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		Errf("Can`t get PVCs %s", nsName)
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

	err = (*clr.rtClient).Create(clr.ctx, &pvc)
	if err != nil {
		return nil, err
		//return coreapi.PersistentVolumeClaim{}, err
	}
	return &pvc, nil
}

func (clr *KCluster) WaitPVCStatus(name string) (string, error) {
	pvc := coreapi.PersistentVolumeClaim{}
	for i := 0; i < PVCWaitIterationCount; i++ {
		err := (*clr.rtClient).Get(clr.ctx, ctrlrtclient.ObjectKey{
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
	// TODO replace name param with Filter
	// func (clr *KCluster) DeletePVC(filters ...Filter) error {
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

	err := (*clr.rtClient).Delete(clr.ctx, &pvc)
	if err != nil {
		return err
	}
	return nil
}

func (clr *KCluster) DeletePVCWait(name string) (string, error) {
	// TODO make implementation
	return "", nil
}

func (clr *KCluster) UpdatePVC(pvc *coreapi.PersistentVolumeClaim) error {
	err := (*clr.rtClient).Update(clr.ctx, pvc)
	if err != nil {
		Errf("Can`t update PVC %s", pvc.Name)
		return err
	}

	return nil
}

/*  Virtual Disk  */

func (clr *KCluster) GetVMD(nsName, vmdName string) ([]virt.VirtualDisk, error) {
	vmds := virt.VirtualDiskList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})

	err := (*clr.rtClient).List(clr.ctx, &vmds, opts)
	if err != nil {
		Errf("Can`t get VMDs (%s, %s)", nsName, vmdName)
		return nil, err
	}

	if vmdName == "" {
		return vmds.Items, nil
	}

	vmdList := []virt.VirtualDisk{}
	for _, vmd := range vmds.Items {
		if vmdName == vmd.Name { // <VM>-data|<VM>-system
			vmdList = append(vmdList, vmd)
		}
	}

	return vmdList, nil
}

func (clr *KCluster) GetTestVMD() ([]virt.VirtualDisk, error) {
	return clr.GetVMD(TestNS, "")
}

func (clr *KCluster) UpdateVMD(vmd *virt.VirtualDisk) error {
	return (*clr.rtClient).Update(clr.ctx, vmd)
}
