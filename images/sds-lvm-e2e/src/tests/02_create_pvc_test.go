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

func TestPVC_23(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	err = funcs.CreatePVC(ctx, cl,
		"test-pvc",
		"test-lvm-thick-immediate-retain",
		"1Gi", false)
	if err != nil {
		t.Error(err)
	}
}

func TestDeletePVC(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	err = funcs.DeletePVC(ctx, cl, "test-pvc-fs")
	if err != nil {
		t.Error(err)
	}
}
