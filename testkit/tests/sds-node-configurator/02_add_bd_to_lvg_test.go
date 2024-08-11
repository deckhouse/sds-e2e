package sds_node_configurator

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddBDtoLVG(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		t.Error("Kubeclient problem", err)
	}

	extCl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("Parent cluster kubeclient problem", err)
	}

	for _, vmName := range []string{"vm1", "vm2", "vm3"} {
		vmdName := fmt.Sprintf("%s-data-add", vmName)
		_, err = funcs.CreateVMD(ctx, extCl, funcs.NamespaceName, vmdName, funcs.StorageClass, 5)
		if err != nil {
			t.Error("Disk creation failed", err)
		}

		err = cl.Create(ctx, &v1alpha2.VirtualMachineBlockDeviceAttachment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmdName,
				Namespace: funcs.NamespaceName,
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
			t.Error("Disk attach failed", err)
		}

	}

}
