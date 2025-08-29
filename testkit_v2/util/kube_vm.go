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

func (cluster *KCluster) ListVM(filters ...VmFilter) ([]vmType, error) {
	vms := virt.VirtualMachineList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})
	err := cluster.controllerRuntimeClient.List(cluster.ctx, &vms, opts)
	if err != nil {
		return nil, err
	}

	resp := vms.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (cluster *KCluster) CreateVM(
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

	vmCVMI, err := cluster.GetClusterVirtualImage(cvmiName)
	if err != nil {
		vmCVMI, err = cluster.CreateClusterVirtualImage(cvmiName, imgUrl)
		if err != nil {
			return fmt.Errorf("CreateClusterVirtualImage: %w", err)
		}
	}

	vmIPClaimName := ""
	if ip != "" {
		vmIPClaim := &virt.VirtualMachineIPAddress{}
		vmIPClaimName = fmt.Sprintf("%s-ipaddress-0", vmName)
		vmIPClaimList, err := cluster.ListIPClaim(nsName, vmIPClaimName)
		if err != nil {
			return err
		}
		if len(vmIPClaimList) == 0 {
			vmIPClaim, err = cluster.CreateVirtualMachineIPAddress(nsName, vmIPClaimName, ip)
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
	if _, err := cluster.GetVD(nsName, vmdName); err != nil {
		vmSystemDisk, err = cluster.CreateVirtualDiskFromClusterVirtualImage(nsName, vmdName, storageClass, systemDriveSize, vmCVMI)
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

	err = cluster.controllerRuntimeClient.Create(cluster.ctx, vmObj)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

/*  Cluster Virtual Image  */

func (cluster *KCluster) GetClusterVirtualImage(cvmiName string) (*virt.ClusterVirtualImage, error) {
	cvmiList, err := cluster.ListClusterVirtualImage()
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

func (cluster *KCluster) ListClusterVirtualImage() ([]virt.ClusterVirtualImage, error) {
	objs := virt.ClusterVirtualImageList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})
	if err := cluster.controllerRuntimeClient.List(cluster.ctx, &objs, opts); err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (cluster *KCluster) CreateClusterVirtualImage(name string, url string) (*virt.ClusterVirtualImage, error) {
	vmCVMI := &virt.ClusterVirtualImage{ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: virt.ClusterVirtualImageSpec{
			DataSource: virt.ClusterVirtualImageDataSource{Type: "HTTP", HTTP: &virt.DataSourceHTTP{URL: url}},
		},
	}

	err := cluster.controllerRuntimeClient.Create(cluster.ctx, vmCVMI)
	if err != nil {
		return nil, err
	}

	return vmCVMI, nil
}

/*  IP  */

func (cluster *KCluster) ListIPClaim(nsName string, vmIPClaimSearch string) ([]virt.VirtualMachineIPAddress, error) {
	objs := virt.VirtualMachineIPAddressList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})
	err := cluster.controllerRuntimeClient.List(cluster.ctx, &objs, opts)
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

func (cluster *KCluster) CreateVirtualMachineIPAddress(
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

	err := cluster.controllerRuntimeClient.Create(cluster.ctx, vmAddr)
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

func (cluster *KCluster) GetVD(nsName, vdName string) (*vdType, error) {
	vd := vdType{}
	err := cluster.controllerRuntimeClient.Get(cluster.ctx, ctrlrtclient.ObjectKey{
		Name:      vdName,
		Namespace: nsName,
	}, &vd)
	if err != nil {
		return nil, err
	}
	return &vd, nil
}

func (cluster *KCluster) ListVD(filters ...VdFilter) ([]vdType, error) {
	vds := virt.VirtualDiskList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})

	err := cluster.controllerRuntimeClient.List(cluster.ctx, &vds, opts)
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

func (cluster *KCluster) CreateVD(name, namespace, storageClass string, sizeInGi int64) error {
	var sc *string = nil
	if storageClass != "" {
		sc = &storageClass
	}

	vmDisk := &virt.VirtualDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: virt.VirtualDiskSpec{
			PersistentVolumeClaim: virt.VirtualDiskPersistentVolumeClaim{
				Size:         resource.NewQuantity(sizeInGi*1024*1024*1024, resource.BinarySI),
				StorageClass: sc,
			},
		},
	}

	err := cluster.controllerRuntimeClient.Create(cluster.ctx, vmDisk)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (cluster *KCluster) UpdateVd(vd *vdType) error {
	err := cluster.controllerRuntimeClient.Update(cluster.ctx, vd)
	if err != nil {
		Errorf("Can't update VD %s", vd.Name)
		return err
	}

	return nil
}

func (cluster *KCluster) DeleteVD(filters ...VdFilter) error {
	vds, err := cluster.ListVD(filters...)
	if err != nil {
		return err
	}

	for _, vd := range vds {
		err := cluster.controllerRuntimeClient.Delete(cluster.ctx, &vd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cluster *KCluster) DeleteVdAndWait(filters ...VdFilter) error {
	if err := cluster.DeleteVD(filters...); err != nil {
		return err
	}

	return RetrySec(15, func() error {
		vds, err := cluster.ListVD(filters...)
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

func (cluster *KCluster) CreateVirtualDiskFromClusterVirtualImage(
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

	err := cluster.controllerRuntimeClient.Create(cluster.ctx, vmDisk)
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

func (f *VmBdFilter) Apply(vmbds []virt.VirtualMachineBlockDeviceAttachment) (resp []virt.VirtualMachineBlockDeviceAttachment) {
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

func (cluster *KCluster) ListVMBD(filters ...VmBdFilter) ([]virt.VirtualMachineBlockDeviceAttachment, error) {
	vmbdas := virt.VirtualMachineBlockDeviceAttachmentList{}
	optsList := ctrlrtclient.ListOptions{}
	opts := ctrlrtclient.ListOption(&optsList)
	if err := cluster.controllerRuntimeClient.List(cluster.ctx, &vmbdas, opts); err != nil {
		return nil, err
	}

	resp := vmbdas.Items
	for _, filter := range filters {
		resp = filter.Apply(resp)
	}

	return resp, nil
}

func (cluster *KCluster) WaitVmbdAttached(filters ...VmBdFilter) error {
	return RetrySec(25, func() error {
		filters = append(filters, VmBdFilter{Phase: "!Attached"})
		vmbds, err := cluster.ListVMBD(filters...)
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

func (cluster *KCluster) AttachVmbd(vmName, vmdName string) error {
	nsName := TestNS
	err := cluster.controllerRuntimeClient.Create(cluster.ctx, &virt.VirtualMachineBlockDeviceAttachment{
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

func (cluster *KCluster) DetachVmbd(filters ...VmBdFilter) error {
	vmbds, err := cluster.ListVMBD(filters...)
	if err != nil {
		return err
	}

	for _, vmbd := range vmbds {
		err := cluster.controllerRuntimeClient.Delete(cluster.ctx, &vmbd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cluster *KCluster) CreateVMBD(vmName, vmdName, storageClassName string, size int64) error {
	nsName := TestNS

	if err := cluster.CreateVD(vmdName, nsName, storageClassName, size); err != nil {
		fmt.Println("err 1")
		return err
	}
	if err := cluster.AttachVmbd(vmName, vmdName); err != nil {
		fmt.Println("err 2")
		return err
	}

	return nil
}

func (cluster *KCluster) DeleteVMBD(filters ...VmBdFilter) error {
	vmbds, err := cluster.ListVMBD(filters...)
	if err != nil {
		return err
	}

	for _, vmbd := range vmbds {
		err := cluster.controllerRuntimeClient.Delete(cluster.ctx, &vmbd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cluster *KCluster) DeleteVmbdAndWait(filters ...VmBdFilter) error {
	if err := cluster.DeleteVMBD(filters...); err != nil {
		return err
	}

	return RetrySec(15, func() error {
		vmbds, err := cluster.ListVMBD(filters...)
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
