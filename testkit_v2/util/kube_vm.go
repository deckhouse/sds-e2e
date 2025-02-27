package integration

import (
	"fmt"
	"strings"
	"time"

	virt "github.com/deckhouse/virtualization/api/core/v1alpha2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlrtclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (clr *KCluster) ListVM(nsName string) ([]virt.VirtualMachine, error) {
	objs := virt.VirtualMachineList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})
	err := clr.rtClient.List(clr.ctx, &objs, opts)
	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (clr *KCluster) CreateVM(
	nsName string,
	vmName string,
	ip string,
	cpu int,
	memory string,
	storageClass string,
	imgName string,
	sshPubKey string,
	systemDriveSize int,
) error {
	imgUrl, ok := Images[imgName]
	if !ok {
		return fmt.Errorf("No '%s' image", imgName)
	}
	cvmiName := strings.ToLower(strings.Replace(strings.Replace(imgName, "_", "-", -1), " ", "-", -1))
	cvmiName = fmt.Sprintf("test-%s-%s", cvmiName, hashMd5(imgUrl)[:4])

	vmCVMI, err := clr.GetCVMI(cvmiName)
	if err != nil {
		vmCVMI, err = clr.CreateCVMI(cvmiName, imgUrl)
		if err != nil {
			return err
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
	if _, err := clr.GetVMD(nsName, vmdName); err != nil {
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

/*  Cluster Virtual Image  */

func (clr *KCluster) GetCVMI(cvmiName string) (*virt.ClusterVirtualImage, error) {
	cvmiList, err := clr.ListCVMI()
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

func (clr *KCluster) ListCVMI() ([]virt.ClusterVirtualImage, error) {
	objs := virt.ClusterVirtualImageList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{})
	if err := clr.rtClient.List(clr.ctx, &objs, opts); err != nil {
		return nil, err
	}

	return objs.Items, nil
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

/*  Virtual Disk (VMD)  */

func (clr *KCluster) GetVMD(nsName, vmdName string) (*virt.VirtualDisk, error) {
	vmdList, err := clr.ListVMD(nsName)
	if err != nil {
		return nil, err
	}

	for _, vmd := range vmdList {
		if vmd.Name == vmdName {
			return &vmd, nil
		}
	}

	return nil, fmt.Errorf("NotFound")
}

func (clr *KCluster) ListVMD(nsName string) ([]virt.VirtualDisk, error) {
	vmds := virt.VirtualDiskList{}
	opts := ctrlrtclient.ListOption(&ctrlrtclient.ListOptions{Namespace: nsName})

	err := clr.rtClient.List(clr.ctx, &vmds, opts)
	if err != nil {
		Debugf("Can't get '%s' VMDs: %s", nsName, err.Error())
		return nil, err
	}

	return vmds.Items, nil
}

func (clr *KCluster) CreateVMD(nsName string, name string, storageClass string, sizeInGi int64) error {
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
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (clr *KCluster) DeleteVMD(nsName, name string) error {
	vmDisk := &virt.VirtualDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nsName,
		},
	}

	if err := clr.rtClient.Delete(clr.ctx, vmDisk); err != nil {
		return err
	}

	return nil
}

func (clr *KCluster) CreateVMDFromCVMI(nsName string, name string, storageClass string, sizeInGi int, vmCVMI *virt.ClusterVirtualImage) (*virt.VirtualDisk, error) {
	vmDisk := &virt.VirtualDisk{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nsName,
		},
		Spec: virt.VirtualDiskSpec{
			PersistentVolumeClaim: virt.VirtualDiskPersistentVolumeClaim{
				Size:         resource.NewQuantity(int64(sizeInGi*1024*1024*1024), resource.BinarySI),
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

/*  VM BlockDevice  */

type VmBdFilter struct {
	NameSpace any
	Name      any
	VmName    any
}

type vmbdType = virt.VirtualMachineBlockDeviceAttachment

func (f *VmBdFilter) Apply(vmbds []vmbdType) (resp []vmbdType) {
	for _, vmbd := range vmbds {
		if f.Name != nil && !CheckCondition(f.Name, vmbd.ObjectMeta.Name) {
			continue
		}
		if f.NameSpace != nil && !CheckCondition(f.Name, vmbd.ObjectMeta.Namespace) {
			continue
		}
		if f.VmName != nil && !CheckCondition(f.VmName, vmbd.Spec.VirtualMachineName) {
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

func (clr *KCluster) WaitVMBD(filters ...VmBdFilter) error {
	for i := 0; ; i++ {
		vmbds, err := clr.ListVMBD(filters...)
		if err != nil {
			return err
		}

		allOk := true
		for _, vmbd := range vmbds {
			if vmbd.Status.Phase != "Attached" {
				allOk = false
				Debugf("VMBD %s not Attached", vmbd.ObjectMeta.Name)
				break
			}
		}
		if allOk {
			break
		}

		if i >= retries {
			Fatalf("Timeout waiting VMBD attached")
		}

		time.Sleep(10 * time.Second)
	}
	return nil
}

func (clr *KCluster) CreateVMBD(vmName, vmdName, storageClass string, size int64) error {
	nsName := TestNS

	if err := clr.CreateVMD(nsName, vmdName, storageClass, size); err != nil {
		return err
	}

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

		err = clr.DeleteVMD(vmbd.ObjectMeta.Namespace, vmbd.ObjectMeta.Name)
		if err != nil {
			return err
		}
	}

	return nil
}
