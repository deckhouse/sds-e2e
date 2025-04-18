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

package funcs

import (
	"context"
	"fmt"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type VM struct {
	Name   string
	Status v1alpha2.MachinePhase
}

type VMD struct {
	Name string
}

type CVMI struct {
	Name string
}

type IPClaim struct {
	Name string
}

func ListVM(ctx context.Context, cl client.Client, namespaceName string) ([]VM, error) {
	objs := v1alpha2.VirtualMachineList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	vmList := []VM{}
	for _, item := range objs.Items {
		vmList = append(vmList, VM{Name: item.Name, Status: item.Status.Phase})
	}

	return vmList, nil
}

func ListVMD(ctx context.Context, cl client.Client, namespaceName string, VMDSearch string) ([]v1alpha2.VirtualDisk, error) {
	objs := v1alpha2.VirtualDiskList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	vmdList := []v1alpha2.VirtualDisk{}
	for _, item := range objs.Items {
		if VMDSearch == "" || VMDSearch == item.Name {
			vmdList = append(vmdList, item)
		}
	}

	return vmdList, nil
}

func ListCVMI(ctx context.Context, cl client.Client, CVMISearch string) ([]CVMI, error) {
	objs := v1alpha2.ClusterVirtualImageList{}
	opts := client.ListOption(&client.ListOptions{})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
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

func ListIPClaim(ctx context.Context, cl client.Client, namespaceName string, vmIPClaimSearch string) ([]v1alpha2.VirtualMachineIPAddress, error) {
	objs := v1alpha2.VirtualMachineIPAddressList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	vmIPClaimList := []v1alpha2.VirtualMachineIPAddress{}
	for _, item := range objs.Items {
		if vmIPClaimSearch == "" || vmIPClaimSearch == item.Name {
			vmIPClaimList = append(vmIPClaimList, item)
		}
	}

	return vmIPClaimList, nil
}

func CreateCVMI(ctx context.Context, cl client.Client, name string, url string) (*v1alpha2.ClusterVirtualImage, error) {
	vmCVMI := &v1alpha2.ClusterVirtualImage{ObjectMeta: metav1.ObjectMeta{
		Name: name,
	},
		Spec: v1alpha2.ClusterVirtualImageSpec{
			DataSource: v1alpha2.ClusterVirtualImageDataSource{Type: "HTTP", HTTP: &v1alpha2.DataSourceHTTP{URL: url}},
		},
	}

	err := cl.Create(ctx, vmCVMI)
	if err != nil {
		return nil, err
	}

	return vmCVMI, nil
}

func CreateVMIPClaim(ctx context.Context, cl client.Client, namespaceName string, name string, ip string) (*v1alpha2.VirtualMachineIPAddress, error) {
	vmClaim := &v1alpha2.VirtualMachineIPAddress{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespaceName,
	},
		Spec: v1alpha2.VirtualMachineIPAddressSpec{
			Type:     v1alpha2.VirtualMachineIPAddressTypeStatic,
			StaticIP: ip,
		},
	}

	err := cl.Create(ctx, vmClaim)
	if err != nil {
		return nil, err
	}

	return vmClaim, nil
}

func CreateVMD(ctx context.Context, cl client.Client, namespaceName string, name string, storageClass string, sizeInGi int64) (*v1alpha2.VirtualDisk, error) {
	vmDisk := &v1alpha2.VirtualDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Spec: v1alpha2.VirtualDiskSpec{
			PersistentVolumeClaim: v1alpha2.VirtualDiskPersistentVolumeClaim{
				Size:         resource.NewQuantity(sizeInGi*1024*1024*1024, resource.BinarySI),
				StorageClass: &storageClass,
			},
		},
	}

	err := cl.Create(ctx, vmDisk)
	if err != nil {
		return nil, err
	}

	return vmDisk, nil
}

func CreateVMDFromCVMI(ctx context.Context, cl client.Client, namespaceName string, name string, storageClass string, sizeInGi int64, vmCVMI *v1alpha2.ClusterVirtualImage) (*v1alpha2.VirtualDisk, error) {
	vmDisk := &v1alpha2.VirtualDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Spec: v1alpha2.VirtualDiskSpec{
			PersistentVolumeClaim: v1alpha2.VirtualDiskPersistentVolumeClaim{
				Size:         resource.NewQuantity(sizeInGi*1024*1024*1024, resource.BinarySI),
				StorageClass: &storageClass,
			},
			DataSource: &v1alpha2.VirtualDiskDataSource{
				Type: v1alpha2.DataSourceTypeObjectRef,
				ObjectRef: &v1alpha2.VirtualDiskObjectRef{
					Kind: v1alpha2.ClusterVirtualImageKind,
					Name: vmCVMI.Name,
				},
			},
		},
	}

	err := cl.Create(ctx, vmDisk)
	if err != nil {
		return nil, err
	}

	return vmDisk, nil
}

func CreateVM(ctx context.Context,
	cl client.Client,
	namespaceName string,
	vmName string,
	ip string,
	cpu int,
	memory string,
	storageClass string,
	url string,
	sshPubKey string,
	systemDriveSize int64,
	dataDriveSize int64) error {

	fmt.Printf("Creating VM %s\n", vmName)

	splittedUrl := strings.Split(url, "/")
	CVMIName := strings.Split(splittedUrl[len(splittedUrl)-1], ".")[0]
	vmCVMI := &v1alpha2.ClusterVirtualImage{}
	CVMIList, err := ListCVMI(ctx, cl, CVMIName)
	if err != nil {
		return err
	}
	if len(CVMIList) == 0 {
		vmCVMI, err = CreateCVMI(ctx, cl, CVMIName, url)
		if err != nil {
			return err
		}
	} else {
		vmCVMI.Name = CVMIList[0].Name
	}

	vmIPClaim := &v1alpha2.VirtualMachineIPAddress{}
	vmIPClaimName := fmt.Sprintf("%s-ipaddress-0", vmName)
	vmIPClaimList, err := ListIPClaim(ctx, cl, namespaceName, vmIPClaimName)
	if err != nil {
		return err
	}
	if len(vmIPClaimList) == 0 {
		vmIPClaim, err = CreateVMIPClaim(ctx, cl, namespaceName, vmIPClaimName, ip)
		if err != nil {
			return err
		}
	} else {
		vmIPClaim = &vmIPClaimList[0]
	}

	vmSystemDisk := &v1alpha2.VirtualDisk{}
	vmdName := fmt.Sprintf("%s-system", vmName)
	vmdList, err := ListVMD(ctx, cl, namespaceName, vmdName)
	if err != nil {
		return err
	}
	if len(vmdList) == 0 {
		vmSystemDisk, err = CreateVMDFromCVMI(ctx, cl, namespaceName, vmdName, storageClass, systemDriveSize, vmCVMI)
		if err != nil {
			return err
		}
	}

	vmdName = fmt.Sprintf("%s-data", vmName)
	vmdList, err = ListVMD(ctx, cl, namespaceName, vmdName)
	if err != nil {
		return err
	}
	if len(vmdList) == 0 {
		_, err := CreateVMD(ctx, cl, namespaceName, vmdName, storageClass, dataDriveSize)
		if err != nil {
			return err
		}
	}

	currentMemory, err := resource.ParseQuantity(memory)
	if err != nil {
		return err
	}

	vmObj := &v1alpha2.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: namespaceName,
			Labels:    map[string]string{"vm": "linux", "service": "v1"},
		},
		Spec: v1alpha2.VirtualMachineSpec{
			VirtualMachineClassName:  "generic",
			EnableParavirtualization: true,
			RunPolicy:                v1alpha2.RunPolicy("AlwaysOn"),
			OsType:                   v1alpha2.OsType("Generic"),
			Bootloader:               v1alpha2.BootloaderType("BIOS"),
			VirtualMachineIPAddress:  vmIPClaim.Name,
			CPU:                      v1alpha2.CPUSpec{Cores: cpu, CoreFraction: "100%"},
			Memory:                   v1alpha2.MemorySpec{Size: currentMemory},
			BlockDeviceRefs: []v1alpha2.BlockDeviceSpecRef{
				{
					Kind: v1alpha2.DiskDevice,
					Name: vmSystemDisk.Name,
				},
			},
			Provisioning: &v1alpha2.Provisioning{
				Type: v1alpha2.ProvisioningType("UserData"),
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

	err = cl.Create(ctx, &v1alpha2.VirtualMachineBlockDeviceAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmdName,
			Namespace: namespaceName,
		},
		Spec: v1alpha2.VirtualMachineBlockDeviceAttachmentSpec{
			VirtualMachineName: vmName,
			BlockDeviceRef: v1alpha2.VMBDAObjectRef{
				Kind: "VirtualDisk",
				Name: vmdName,
			},
		},
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}

	return cl.Create(ctx, vmObj)
}
