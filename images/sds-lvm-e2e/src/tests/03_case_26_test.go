package test

import (
	"context"
	"errors"
	"fmt"
	"sds-lvm-e2e/funcs"
	"testing"
)

func init() {
	fmt.Println("Create manual LVG resource vg-data-on-node-worker-1")
}

func Test_26(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	pvcName, err := funcs.CreatePVC(ctx, cl,
		"test-pvc", "test-lvm-thick-immediate-retain", "1Gi", false)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("pvc=%s created", pvcName))

	command := []string{"/bin/bash"}
	args := []string{"-c", "touch /usr/share/test-data/test.txt"}
	podName, err := funcs.CreatePod(ctx, cl, "test-pod", pvcName, false, command, args)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("pod=%s created", podName))

	status, err := funcs.WaitPodStatus(ctx, cl, podName)
	if err != nil {
		t.Error(err)
	}

	t.Log(fmt.Sprintf("status pod=%s ", status))
	if status == "Error" {
		t.Error(errors.New("container error"))
	}

	err = funcs.DeletePod(ctx, cl, podName)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("pod=%s deleted", podName))

	err = funcs.DeletePVC(ctx, cl, pvcName)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("pvc=%s deleted", pvcName))
}
