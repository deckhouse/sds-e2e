package test

import (
	"context"
	"fmt"
	"sds-lvm-e2e/funcs"
	"testing"
)

func init() {
	fmt.Println("Create manual LVG resource vg-data-on-node-worker-1")
}

func Test_27(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	t.Log("------------  pvc creating ------------- ")
	pvc, err := funcs.CreatePVC(ctx, cl,
		"test-pvc", "test-lvm-thick-immediate-retain", "1Gi", false)
	if err != nil {
		t.Error(err)
	}
	pvcStatus, err := funcs.WaitPVCStatus(ctx, cl, pvc.Name)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("pvc status=%s", pvcStatus))
	t.Log(fmt.Sprintf("pvc=%s created", pvc.Name))

	t.Log("------------  pod creating ------------- ")
	command := []string{"/bin/bash"}
	args := []string{"-c", "touch /usr/share/test-data/test.txt"}
	podName, err := funcs.CreatePod(ctx, cl, "test-pod", pvc.Name, false, command, args)
	if err != nil {
		t.Error(err)
	}
	status, err := funcs.WaitPodStatus(ctx, cl, podName)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("status pod=%s ", status))
	t.Log(fmt.Sprintf("pod=%s created", podName))
	t.Log(fmt.Sprintf("exec command=%s ", args[1]))

	t.Log("------------  pod deleting ------------- ")
	err = funcs.DeletePod(ctx, cl, podName)
	if err != nil {
		t.Error(err)
	}
	deleteStatus, err := funcs.WaitDeletePod(ctx, cl, podName)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("pod delete status=%s", deleteStatus))
	t.Log(fmt.Sprintf("pod=%s deleted", podName))

	t.Log("------------  pod creating ------------- ")
	command = []string{"/bin/bash"}
	args = []string{"-c", "cat /usr/share/test-data/test.txt"}
	podName, err = funcs.CreatePod(ctx, cl, "test-pod", pvc.Name, false, command, args)
	if err != nil {
		t.Error(err)
	}
	status, err = funcs.WaitPodStatus(ctx, cl, podName)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("status pod=%s ", status))
	t.Log(fmt.Sprintf("pod=%s created", podName))
	t.Log(fmt.Sprintf("exec command=%s ", args[1]))

	t.Log("------------  pod deleting ------------- ")
	err = funcs.DeletePod(ctx, cl, podName)
	if err != nil {
		t.Error(err)
	}
	deleteStatus, err = funcs.WaitDeletePod(ctx, cl, podName)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("pod delete status=%s", deleteStatus))
	t.Log(fmt.Sprintf("pod=%s deleted", podName))

	t.Log("------------  pvc deleting ------------- ")
	err = funcs.DeletePVC(ctx, cl, pvc.Name)
	if err != nil {
		t.Error(err)
	}
	pvcStatus, err = funcs.WaitPVCStatus(ctx, cl, pvc.Name)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("pvc status=%s", pvcStatus))
	t.Log(fmt.Sprintf("pvc=%s deleted", pvc.Name))
}
