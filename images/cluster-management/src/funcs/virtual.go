package funcs

import (
	"cluster-management/v1alpha2"
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type VM struct {
	Name string
}

type CVMI struct {
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

func ListCVMI(ctx context.Context, cl client.Client) ([]CVMI, error) {
	objs := v1alpha2.ClusterVirtualMachineImageList{}
	opts := client.ListOption(&client.ListOptions{})
	err := cl.List(ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	cvmiList := []CVMI{}
	for _, item := range objs.Items {
		cvmiList = append(cvmiList, CVMI{Name: item.Name})
	}

	return cvmiList, nil
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
	CVMIList, err := ListCVMI(ctx, cl)
	if err != nil {
		return err
	}

	CVMIExists := false
	for _, CVMI := range CVMIList {
		if CVMI.Name == CVMIName {
			CVMIExists = true
			break
		}
	}

	fmt.Printf("CVMI Exists: %v\n", CVMIExists)
	os.Exit(1)

	if !CVMIExists {
		_, err := CreateCVMI(ctx, cl, CVMIName, url)
		if err != nil {
			fmt.Println(err.Error() != fmt.Sprintf("clustervirtualmachineimages.virtualization.deckhouse.io \"%s\" already exists", CVMIName))
			if err.Error() != fmt.Sprintf("clustervirtualmachineimages.virtualization.deckhouse.io \"%s\" already exists", CVMIName) {
				return err
			}
		}
	}

	print(1)

	vmClaim, err := CreateVMIPClaim(ctx, cl, namespaceName, fmt.Sprintf("%s-0", vmName), ip)
	if err != nil {
		return err
	}

	vmSystemDisk, err := CreateVMD(ctx, cl, namespaceName, fmt.Sprintf("%s-system", vmName), storageClass, 32)
	if err != nil {
		return err
	}

	vmDataDisk, err := CreateVMD(ctx, cl, namespaceName, fmt.Sprintf("%s-data", vmName), storageClass, 50)
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
			RunPolicy:                        v1alpha2.RunPolicy("AlwaysOn"),
			OsType:                           v1alpha2.OsType("Generic"),
			Bootloader:                       v1alpha2.BootloaderType("BIOS"),
			VirtualMachineIPAddressClaimName: vmClaim.Name,
			CPU:                              v1alpha2.CPUSpec{Cores: cpu, CoreFraction: "100%"},
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
		},
	}

	return cl.Create(ctx, vmObj)
}
