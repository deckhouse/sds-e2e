package integration

import (
	"fmt"
	"time"

	snc "github.com/deckhouse/sds-node-configurator/api/v1alpha1"
	coreapi "k8s.io/api/core/v1"
	storapi "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

/*  Block Device  */

type BdFilter struct {
	Name       any
	Node       any
	Consumable any
	Size       float32
}

type bdType = snc.BlockDevice

func (f *BdFilter) Apply(bds []bdType) (resp []bdType) {
	for _, bd := range bds {
		if f.Name != nil && !CheckCondition(f.Name, bd.Name) {
			continue
		}
		if f.Node != nil && !CheckCondition(f.Node, bd.Status.NodeName) {
			continue
		}
		if f.Consumable != nil && !CheckCondition(f.Consumable, bd.Status.Consumable) {
			continue
		}
		if f.Size != 0 {
			s := int64(f.Size * 1024 * 1024 * 1024)
			if bd.Status.Size.Value() < s || bd.Status.Size.Value() > s+10737418 {
				continue
			}
		}

		resp = append(resp, bd)
	}
	return
}

func (clr *KCluster) ListBD(filters ...BdFilter) ([]bdType, error) {
	bdList := &snc.BlockDeviceList{}
	err := clr.rtClient.List(clr.ctx, bdList)
	if err != nil {
		Warnf("Can't get BDs: %s", err.Error())
		return nil, err
	}

	resp := bdList.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (clr *KCluster) DeleteBd(filters ...BdFilter) error {
	bdList, err := clr.ListBD(filters...)
	if err != nil {
		return err
	}
	for _, bd := range bdList {
		if err := clr.rtClient.Delete(clr.ctx, &bd); err != nil {
			return err
		}
	}

	return nil
}

func (clr *KCluster) DeleteBdWithCheck(filters ...BdFilter) error {
	if err := clr.DeleteBd(filters...); err != nil {
		return err
	}
	return RetrySec(20, func() error {
		bds, err := clr.ListBD(filters...)
		if err != nil {
			return err
		}

		if len(bds) > 0 {
			return fmt.Errorf("not deleted BDs: %d (%s, ...)", len(bds), bds[0].Name)
		}
		return nil
	})
}

/*  LVM Volume Group  */

type lvgType = snc.LVMVolumeGroup

type LvgFilter struct {
	Name  any
	Node  any
	Phase any
}

func (f *LvgFilter) Apply(lvgs []lvgType) (resp []lvgType) {
	for _, lvg := range lvgs {
		if f.Name != nil && !CheckCondition(f.Name, lvg.Name) {
			continue
		}
		if f.Node != nil && (len(lvg.Status.Nodes) == 0 || !CheckCondition(f.Node, lvg.Status.Nodes[0].Name)) {
			continue
		}
		if f.Phase != nil && !CheckCondition(f.Phase, lvg.Status.Phase) {
			continue
		}

		resp = append(resp, lvg)
	}
	return
}

func (clr *KCluster) GetLvg(lvgName string) (*lvgType, error) {
	lvg := lvgType{}
	err := clr.rtClient.Get(clr.ctx, ctrlrtclient.ObjectKey{
		Name: lvgName,
	}, &lvg)
	if err != nil {
		return nil, err
	}
	return &lvg, nil
}

func (clr *KCluster) ListLVG(filters ...LvgFilter) ([]lvgType, error) {
	lvgList := &snc.LVMVolumeGroupList{}
	if err := clr.rtClient.List(clr.ctx, lvgList); err != nil {
		Warnf("Can't get LVGs: %s", err.Error())
		return nil, err
	}

	resp := lvgList.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (clr *KCluster) CheckLVGsReady(filters ...LvgFilter) error {
	filtersNotReady := append(filters, LvgFilter{Phase: "!Ready"})
	if err := RetrySec(35, func() error {
		lvgs, err := clr.ListLVG(filtersNotReady...)
		if err != nil {
			return err
		}
		if len(lvgs) > 0 {
			return fmt.Errorf("LVGs not Ready: %d (%s, ...)", len(lvgs), lvgs[0].Name)
		}
		return nil
	}); err != nil {
		return err
	}

	lvgs, err := clr.ListLVG(filters...)
	if err != nil {
		return err
	}
	for _, lvg := range lvgs {
		if len(lvg.Status.Nodes) == 0 {
			return fmt.Errorf("no nodes in LVG %s status", lvg.Name)
		}
	}

	return nil
}

func (clr *KCluster) CreateLVG(name, nodeName string, bds []string) error {
	lvmVolumeGroup := &snc.LVMVolumeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: snc.LVMVolumeGroupSpec{
			ActualVGNameOnTheNode: name,
			BlockDeviceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{Key: "kubernetes.io/metadata.name", Operator: metav1.LabelSelectorOpIn, Values: bds},
				},
			},
			Type:  "Local",
			Local: snc.LVMVolumeGroupLocalSpec{NodeName: nodeName},
		},
	}
	err := clr.rtClient.Create(clr.ctx, lvmVolumeGroup)
	if err != nil {
		Errf("Can't create LVG %s (node %s, bds %v)", name, nodeName, bds)
		return err
	}
	return nil
}

func (clr *KCluster) CreateLvgWithCheck(name, nodeName string, bds []string) error {
	if err := clr.CreateLVG(name, nodeName, bds); err != nil {
		return err
	}

	return clr.CheckLVGsReady(LvgFilter{Name: name})
}

func (clr *KCluster) CreateLvgExt(name, nodeName string, ext map[string]any) error {
	lvg := &snc.LVMVolumeGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: snc.LVMVolumeGroupSpec{
			ActualVGNameOnTheNode: name,
			BlockDeviceSelector:   &metav1.LabelSelector{},
			Type:                  "Local",
			Local:                 snc.LVMVolumeGroupLocalSpec{NodeName: nodeName},
		},
	}
	if bds, ok := ext["bds"]; ok {
		lvg.Spec.BlockDeviceSelector.MatchExpressions = []metav1.LabelSelectorRequirement{{
			Key:      "kubernetes.io/metadata.name",
			Operator: metav1.LabelSelectorOpIn,
			Values:   bds.([]string),
		}}
	}
	if tp, ok := ext["thinpools"]; ok {
		lvg.Spec.ThinPools = tp.([]snc.LVMVolumeGroupThinPoolSpec)
	}
	err := clr.rtClient.Create(clr.ctx, lvg)
	if err != nil {
		Errf("Can't create LVG %s/%s", nodeName, name)
		return err
	}
	return nil
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
	lvgs, _ := clr.ListLVG(filters...)

	for _, lvg := range lvgs {
		if err := clr.rtClient.Delete(clr.ctx, &lvg); err != nil {
			return err
		}
	}

	return nil
}

func (clr *KCluster) DeleteLvgWithCheck(filters ...LvgFilter) error {
	if err := clr.DeleteLVG(filters...); err != nil {
		return err
	}

	return RetrySec(15, func() error {
		lvgs, err := clr.ListLVG(filters...)
		if err != nil {
			return err
		}
		if len(lvgs) > 0 {
			return fmt.Errorf("LVGs not deleted: %d", len(lvgs))
		}
		return nil
	})
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

/*  Persistent Volume Claims  */

func (clr *KCluster) ListPVC(nsName string) ([]coreapi.PersistentVolumeClaim, error) {
	pvcList, err := (*clr.goClient).CoreV1().PersistentVolumeClaims(nsName).List(clr.ctx, metav1.ListOptions{})
	if err != nil {
		Debugf("Can't get '%s' PVCs: %s", nsName, err.Error())
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
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
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
	for i := 0; i < pvcWaitIterationCount; i++ {
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
			return "Deleted", nil
		}

		time.Sleep(pvcWaitInterval * time.Second)
	}
	return string(pvc.Status.Phase), fmt.Errorf("the waiting time %d or the pvc to be ready has expired",
		pvcWaitInterval*pvcWaitIterationCount)
}

func (clr *KCluster) DeletePVC(name string) error {
	pvc := coreapi.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
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

func (clr *KCluster) UpdatePVC(pvc *coreapi.PersistentVolumeClaim) error {
	err := clr.rtClient.Update(clr.ctx, pvc)
	if err != nil {
		Warnf("Can't update PVC %s", pvc.Name)
		return err
	}

	return nil
}
