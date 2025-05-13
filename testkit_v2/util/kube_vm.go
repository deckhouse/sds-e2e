/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"fmt"
	"strings"

	virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type vmType = virt.VirtualMachine

type VmFilter struct {
	NameSpace any
	Name      any
	Phase     any
}

func (f *VmFilter) Apply(vms []vmType) (resp []vmType) {
	for _, vm := range vms {
		if f.Name != nil && !CheckCondition(f.Name, vm.Name) {
			continue
		}
		if f.NameSpace != nil && !CheckCondition(f.NameSpace, vm.Namespace) {
			continue
		}
		if f.Phase != nil && !CheckCondition(f.Phase, string(vm.Status.Phase)) {
			continue
		}
		resp = append(resp, vm)
	}
	return
}

func (clr *KCluster) ListVM(filters ...VmFilter) ([]vmType, error) {
	vms := virt.VirtualMachineList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})
	err := clr.rtClient.List(clr.ctx, &vms, opts)
	if err != nil {
		return nil, err
	}

	resp := vms.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (clr *KCluster) CreateVM(
	nsName, vmName, ip string,
	cpu, ram int,
	storageClass, image, sshPubKey string,
	systemDriveSize int,
) error {
	cvmiName := "noname"
	imgUrl, ok := Images[image]
	if !ok {
		imgUrl = image
	} else {
		cvmiName = image
	}
	cvmiName = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(cvmiName, "_", "-"), " ", "-"))
	cvmiName = fmt.Sprintf("test-%s-%s", cvmiName, hashMd5(imgUrl)[:4])

	vmCVMI, err := clr.GetClusterVirtualImage(cvmiName)
	if err != nil {
		vmCVMI, err = clr.CreateClusterVirtualImage(cvmiName, imgUrl)
		if err != nil {
			return fmt.Errorf("CreateClusterVirtualImage: %w", err)
		}
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
			vmIPClaim, err = clr.CreateVirtualMachineIPAddress(nsName, vmIPClaimName, ip)
			if err != nil {
				return fmt.Errorf("CreateVirtualMachineIPAddress: %w", err)
			}
		} else {
			vmIPClaim = &vmIPClaimList[0]
		}
		vmIPClaimName = vmIPClaim.Name
	}

	vmSystemDisk := &virt.VirtualDisk{}
	vmdName := fmt.Sprintf("%s-system", vmName)
	if _, err := clr.GetVD(nsName, vmdName); err != nil {
		vmSystemDisk, err = clr.CreateVirtualDiskFromClusterVirtualImage(nsName, vmdName, storageClass, systemDriveSize, vmCVMI)
		if err != nil {
			return fmt.Errorf("CreateVirtualDiskFromClusterVirtualImage: %w", err)
		}
	}

	currentMemory, err := resource.ParseQuantity(fmt.Sprintf("%dGi", ram))
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

/*  Cluster Virtual Image  */

func (clr *KCluster) GetClusterVirtualImage(cvmiName string) (*virt.ClusterVirtualImage, error) {
	cvmiList, err := clr.ListClusterVirtualImage()
	if err != nil {
		return nil, err
	}

	for _, cvmi := range cvmiList {
		if cvmiName == cvmi.Name {
			return &cvmi, nil
		}
	}

	return nil, fmt.Errorf("NotFound")
}

func (clr *KCluster) ListClusterVirtualImage() ([]virt.ClusterVirtualImage, error) {
	objs := virt.ClusterVirtualImageList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})
	if err := clr.rtClient.List(clr.ctx, &objs, opts); err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (clr *KCluster) CreateClusterVirtualImage(name string, url string) (*virt.ClusterVirtualImage, error) {
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

/*  IP  */

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

func (clr *KCluster) CreateVirtualMachineIPAddress(
	nsName, name, ip string,
) (*virt.VirtualMachineIPAddress, error) {
	vmAddr := &virt.VirtualMachineIPAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nsName,
		},
		Spec: virt.VirtualMachineIPAddressSpec{
			Type:     virt.VirtualMachineIPAddressTypeStatic,
			StaticIP: ip,
		},
	}

	err := clr.rtClient.Create(clr.ctx, vmAddr)
	if err != nil {
		return nil, err
	}

	return vmAddr, nil
}

/*  Virtual Disk (VD)  */

type vdType = virt.VirtualDisk

type VdFilter struct {
	NameSpace any
	Name      any
	Phase     any
}

func (f *VdFilter) Apply(vds []vdType) (resp []vdType) {
	for _, vd := range vds {
		if f.Name != nil && !CheckCondition(f.Name, vd.Name) {
			continue
		}
		if f.NameSpace != nil && !CheckCondition(f.NameSpace, vd.Namespace) {
			continue
		}
		if f.Phase != nil && !CheckCondition(f.Phase, string(vd.Status.Phase)) {
			continue
		}
		resp = append(resp, vd)
	}
	return
}

func (clr *KCluster) GetVD(nsName, vdName string) (*vdType, error) {
	vd := vdType{}
	err := clr.rtClient.Get(clr.ctx, ctrlrtclient.ObjectKey{
		Name:      vdName,
		Namespace: nsName,
	}, &vd)
	if err != nil {
		return nil, err
	}
	return &vd, nil
}

func (clr *KCluster) ListVD(filters ...VdFilter) ([]vdType, error) {
	vds := virt.VirtualDiskList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})

	err := clr.rtClient.List(clr.ctx, &vds, opts)
	if err != nil {
		Debugf("Can't get VDs: %s", err.Error())
		return nil, err
	}

	resp := vds.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (clr *KCluster) CreateVD(nsName string, name string, storageClass string, sizeInGi int64) error {
	var sc *string = nil
	if storageClass != "" {
		sc = &storageClass
	}

	vmDisk := &virt.VirtualDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nsName,
		},
		Spec: virt.VirtualDiskSpec{
			PersistentVolumeClaim: virt.VirtualDiskPersistentVolumeClaim{
				Size:         resource.NewQuantity(sizeInGi*1024*1024*1024, resource.BinarySI),
				StorageClass: sc,
			},
		},
	}

	err := clr.rtClient.Create(clr.ctx, vmDisk)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (clr *KCluster) UpdateVd(vd *vdType) error {
	err := clr.rtClient.Update(clr.ctx, vd)
	if err != nil {
		Errorf("Can't update VD %s", vd.Name)
		return err
	}

	return nil
}

func (clr *KCluster) DeleteVD(filters ...VdFilter) error {
	vds, err := clr.ListVD(filters...)
	if err != nil {
		return err
	}

	for _, vd := range vds {
		err := clr.rtClient.Delete(clr.ctx, &vd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (clr *KCluster) DeleteVdWithCheck(filters ...VdFilter) error {
	if err := clr.DeleteVD(filters...); err != nil {
		return err
	}

	return RetrySec(15, func() error {
		vds, err := clr.ListVD(filters...)
		if err != nil {
			return err
		}
		if len(vds) > 0 {
			return fmt.Errorf("VDs not deleted: %d", len(vds))
		}
		Debugf("VDs deleted")
		return nil
	})
}

func (clr *KCluster) CreateVirtualDiskFromClusterVirtualImage(
	nsName, name, storageClass string,
	sizeInGi int,
	vmCVMI *virt.ClusterVirtualImage,
) (*virt.VirtualDisk, error) {
	var sc *string = nil
	if storageClass != "" {
		sc = &storageClass
	}

	vmDisk := &virt.VirtualDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nsName,
		},
		Spec: virt.VirtualDiskSpec{
			PersistentVolumeClaim: virt.VirtualDiskPersistentVolumeClaim{
				Size:         resource.NewQuantity(int64(sizeInGi)*1024*1024*1024, resource.BinarySI),
				StorageClass: sc,
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

/*  VM BlockDevice  */

type VmBdFilter struct {
	NameSpace any
	Name      any
	VmName    any
	VdName    any
	Phase     any
}

type vmbdType = virt.VirtualMachineBlockDeviceAttachment

func (f *VmBdFilter) Apply(vmbds []vmbdType) (resp []vmbdType) {
	for _, vmbd := range vmbds {
		if f.Name != nil && !CheckCondition(f.Name, vmbd.Name) {
			continue
		}
		if f.NameSpace != nil && !CheckCondition(f.Name, vmbd.Namespace) {
			continue
		}
		if f.VmName != nil && !CheckCondition(f.VmName, vmbd.Spec.VirtualMachineName) {
			continue
		}
		if f.VdName != nil && !CheckCondition(f.VdName, vmbd.Spec.BlockDeviceRef.Name) {
			continue
		}
		if f.Phase != nil && !CheckCondition(f.Phase, string(vmbd.Status.Phase)) {
			continue
		}
		resp = append(resp, vmbd)
	}
	return
}

func (clr *KCluster) ListVMBD(filters ...VmBdFilter) ([]vmbdType, error) {
	vmbdas := virt.VirtualMachineBlockDeviceAttachmentList{}
	optsList := ctrlrtclient.ListOptions{}
	opts := ctrlrtclient.ListOption(&optsList)
	if err := clr.rtClient.List(clr.ctx, &vmbdas, opts); err != nil {
		return nil, err
	}

	resp := vmbdas.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (clr *KCluster) WaitVmbdAttached(filters ...VmBdFilter) error {
	return RetrySec(25, func() error {
		filters = append(filters, VmBdFilter{Phase: "!Attached"})
		vmbds, err := clr.ListVMBD(filters...)
		if err != nil {
			return err
		}
		if len(vmbds) > 0 {
			return fmt.Errorf("VMBDs not Attached: %d (%s, ...)", len(vmbds), vmbds[0].Name)
		}
		Debugf("VMBDs attached")
		return nil
	})
}

func (clr *KCluster) AttachVmbd(vmName, vmdName string) error {
	nsName := TestNS
	err := clr.rtClient.Create(clr.ctx, &virt.VirtualMachineBlockDeviceAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmdName,
			Namespace: nsName,
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

func (clr *KCluster) DetachVmbd(filters ...VmBdFilter) error {
	vmbds, err := clr.ListVMBD(filters...)
	if err != nil {
		return err
	}

	for _, vmbd := range vmbds {
		err := clr.rtClient.Delete(clr.ctx, &vmbd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (clr *KCluster) CreateVMBD(vmName, vmdName, storageClass string, size int64) error {
	nsName := TestNS

	if err := clr.CreateVD(nsName, vmdName, storageClass, size); err != nil {
		return err
	}
	if err := clr.AttachVmbd(vmName, vmdName); err != nil {
		return err
	}

	return nil
}

func (clr *KCluster) DeleteVMBD(filters ...VmBdFilter) error {
	vmbds, err := clr.ListVMBD(filters...)
	if err != nil {
		return err
	}

	for _, vmbd := range vmbds {
		err := clr.rtClient.Delete(clr.ctx, &vmbd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (clr *KCluster) DeleteVmbdWithCheck(filters ...VmBdFilter) error {
	if err := clr.DeleteVMBD(filters...); err != nil {
		return err
	}

	return RetrySec(15, func() error {
		vmbds, err := clr.ListVMBD(filters...)
		if err != nil {
			return err
		}
		if len(vmbds) > 0 {
			return fmt.Errorf("VMBDs not deleted: %d", len(vmbds))
		}
		Debugf("VMBDs deleted")
		return nil
	})
}
