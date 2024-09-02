package sds_node_configurator

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"
)

func TestDeleteThinDataDisks(t *testing.T) {
	ctx := context.Background()

	extCl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("Parent cluster kubeclient problem", err)
	}

	listDataDisksAttachments := &v1alpha2.VirtualMachineBlockDeviceAttachmentList{}
	err = extCl.List(ctx, listDataDisksAttachments, &client.ListOptions{Namespace: funcs.NamespaceName})
	if err != nil {
		t.Error("error listing vmdba", err)
	}

	for _, attachment := range listDataDisksAttachments.Items {
		err = extCl.Delete(ctx, &attachment)
		if err != nil {
			t.Error("error deleting vmdba", err)
		}
	}

	listDataDisks := &v1alpha2.VirtualDiskList{}
	err = extCl.List(ctx, listDataDisks, &client.ListOptions{Namespace: funcs.NamespaceName})
	if err != nil {
		t.Error("error listing vd", err)
	}

	for _, disk := range listDataDisks.Items {
		if strings.Contains(disk.Name, "-thindata") {
			err = extCl.Delete(ctx, &disk)
			if err != nil {
				t.Error("error deleting vd", err)
			}
		}
	}
}
