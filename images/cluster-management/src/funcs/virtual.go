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
	Name string
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
		vmList = append(vmList, VM{Name: item.Name})
	}

	return vmList, nil
}

func ListVMD(ctx context.Context, cl client.Client, namespaceName string, VMDSearch string) ([]VMD, error) {
	objs := v1alpha2.VirtualMachineDiskList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	vmdList := []VMD{}
	for _, item := range objs.Items {
		if VMDSearch == "" || VMDSearch == item.Name {
			vmdList = append(vmdList, VMD{Name: item.Name})
		}
	}

	return vmdList, nil
}

func ListCVMI(ctx context.Context, cl client.Client, CVMISearch string) ([]CVMI, error) {
	objs := v1alpha2.ClusterVirtualMachineImageList{}
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

func ListIPClaim(ctx context.Context, cl client.Client, namespaceName string, vmIPClaimSearch string) ([]IPClaim, error) {
	objs := v1alpha2.VirtualMachineIPAddressClaimList{}
	opts := client.ListOption(&client.ListOptions{Namespace: namespaceName})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	vmIPClaimList := []IPClaim{}
	for _, item := range objs.Items {
		if vmIPClaimSearch == "" || vmIPClaimSearch == item.Name {
			vmIPClaimList = append(vmIPClaimList, IPClaim{Name: item.Name})
		}
	}

	return vmIPClaimList, nil
}

func CreateCVMI(ctx context.Context, cl client.Client, name string, url string) (*v1alpha2.ClusterVirtualMachineImage, error) {
	vmCVMI := &v1alpha2.ClusterVirtualMachineImage{ObjectMeta: metav1.ObjectMeta{
		Name: name,
	},
		Spec: v1alpha2.ClusterVirtualMachineImageSpec{
			DataSource: v1alpha2.CVMIDataSource{Type: "HTTP", HTTP: &v1alpha2.DataSourceHTTP{URL: url}},
		},
	}

	err := cl.Create(ctx, vmCVMI)
	if err != nil {
		return nil, err
	}

	return vmCVMI, nil
}

func CreateVMIPClaim(ctx context.Context, cl client.Client, namespaceName string, name string, ip string) (*v1alpha2.VirtualMachineIPAddressClaim, error) {
	vmClaim := &v1alpha2.VirtualMachineIPAddressClaim{ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespaceName,
	},
		Spec: v1alpha2.VirtualMachineIPAddressClaimSpec{
			Address:       ip,
			ReclaimPolicy: "Delete",
		},
	}

	err := cl.Create(ctx, vmClaim)
	if err != nil {
		return nil, err
	}

	return vmClaim, nil
}

func CreateVMD(ctx context.Context, cl client.Client, namespaceName string, name string, storageClass string, sizeInGi int64) (*v1alpha2.VirtualMachineDisk, error) {
	vmDisk := &v1alpha2.VirtualMachineDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Spec: v1alpha2.VirtualMachineDiskSpec{
			PersistentVolumeClaim: v1alpha2.VMDPersistentVolumeClaim{
				Size:             resource.NewQuantity(sizeInGi*1024*1024*1024, resource.BinarySI),
				StorageClassName: &storageClass,
			},
		},
	}

	err := cl.Create(ctx, vmDisk)
	if err != nil {
		return nil, err
	}

	return vmDisk, nil
}

func CreateVMDFromCVMI(ctx context.Context, cl client.Client, namespaceName string, name string, storageClass string, sizeInGi int64, vmCVMI *v1alpha2.ClusterVirtualMachineImage) (*v1alpha2.VirtualMachineDisk, error) {
	vmDisk := &v1alpha2.VirtualMachineDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespaceName,
		},
		Spec: v1alpha2.VirtualMachineDiskSpec{
			PersistentVolumeClaim: v1alpha2.VMDPersistentVolumeClaim{
				Size:             resource.NewQuantity(sizeInGi*1024*1024*1024, resource.BinarySI),
				StorageClassName: &storageClass,
			},
			DataSource: &v1alpha2.VMDDataSource{
				Type: v1alpha2.DataSourceTypeClusterVirtualMachineImage,
				ClusterVirtualMachineImage: &v1alpha2.DataSourceNamedRef{
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
	url string) error {

	fmt.Printf("Creating VM %s\n", vmName)

	splittedUrl := strings.Split(url, "/")
	CVMIName := strings.Split(splittedUrl[len(splittedUrl)-1], ".")[0]
	vmCVMI := &v1alpha2.ClusterVirtualMachineImage{}
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

	vmIPClaim := &v1alpha2.VirtualMachineIPAddressClaim{}
	vmIPClaimName := fmt.Sprintf("%s-0", vmName)
	vmIPClaimList, err := ListIPClaim(ctx, cl, namespaceName, vmIPClaimName)
	if err != nil {
		return err
	}
	if len(vmIPClaimList) == 0 {
		vmIPClaim, err = CreateVMIPClaim(ctx, cl, namespaceName, vmIPClaimName, ip)
		if err != nil {
			return err
		}
	}

	vmSystemDisk := &v1alpha2.VirtualMachineDisk{}
	vmdName := fmt.Sprintf("%s-system", vmName)
	vmdList, err := ListVMD(ctx, cl, namespaceName, vmdName)
	if err != nil {
		return err
	}
	if len(vmdList) == 0 {
		vmSystemDisk, err = CreateVMDFromCVMI(ctx, cl, namespaceName, vmdName, storageClass, 32, vmCVMI)
		if err != nil {
			return err
		}
	}

	vmDataDisk := &v1alpha2.VirtualMachineDisk{}
	vmdName = fmt.Sprintf("%s-data", vmName)
	vmdList, err = ListVMD(ctx, cl, namespaceName, vmdName)
	if err != nil {
		return err
	}
	if len(vmdList) == 0 {
		vmDataDisk, err = CreateVMD(ctx, cl, namespaceName, vmdName, storageClass, 32)
		if err != nil {
			return err
		}
	}

	vmObj := &v1alpha2.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmName,
			Namespace: namespaceName,
			Labels:    map[string]string{"vm": "linux", "service": "v1"},
		},
		Spec: v1alpha2.VirtualMachineSpec{
			EnableParavirtualization:         true,
			RunPolicy:                        v1alpha2.RunPolicy("AlwaysOn"),
			OsType:                           v1alpha2.OsType("Generic"),
			Bootloader:                       v1alpha2.BootloaderType("BIOS"),
			VirtualMachineIPAddressClaimName: vmIPClaim.Name,
			CPU:                              v1alpha2.CPUSpec{Cores: cpu, CoreFraction: "100%", ModelName: "generic-v1"},
			Memory:                           v1alpha2.MemorySpec{Size: memory},
			BlockDevices: []v1alpha2.BlockDeviceSpec{
				{
					Type:               v1alpha2.DiskDevice,
					VirtualMachineDisk: &v1alpha2.DiskDeviceSpec{Name: vmSystemDisk.Name},
				},
				{
					Type:               v1alpha2.DiskDevice,
					VirtualMachineDisk: &v1alpha2.DiskDeviceSpec{Name: vmDataDisk.Name},
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
`, vmName),
			},
		},
	}

	return cl.Create(ctx, vmObj)
}
