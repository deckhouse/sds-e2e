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

func Test_23_24(t *testing.T) {
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
	args := []string{"-c", "df -T | grep '/usr/share/test-data' | grep 'ext4'"}
	podName, err := funcs.CreatePod(ctx, cl, "test-pod", pvc.Name, false, command, args)
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

	t.Log("------------  pod deleting ------------- ")
	err = funcs.DeletePod(ctx, cl, podName)
	if err != nil {
		t.Error(err)
	}
	status, err = funcs.WaitDeletePod(ctx, cl, podName)
	if err != nil {
		t.Error(err)
	}
	t.Log(fmt.Sprintf("status pod=%s ", status))
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
