package test

import (
	"context"
	"fmt"
	"sds-lvm-e2e/funcs"
	"testing"
)

func init() {
	fmt.Println("Create manual LVM VolumeGroup: vg-w1 in the node")
}

func TestCreateStorageClass_17(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	err = funcs.CreateStorageClass(
		ctx, cl,
		"test-lvm-thick-immediate-delete", "Thick", "- name: vg-w1\n- name: vg-w2",
		funcs.StorageClassVolumeBindingModeImmediate, funcs.StorageClassReclaimPolicyDelete)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateStorageClass_18(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	err = funcs.CreateStorageClass(
		ctx, cl,
		"test-lvm-thick-immediate-retain", "Thick", "- name: vg-w1\n- name: vg-w2",
		funcs.StorageClassVolumeBindingModeImmediate, funcs.StorageClassReclaimPolicyRetain)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateStorageClass_19(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	err = funcs.CreateStorageClass(
		ctx, cl,
		"test-lvm-thick-wait-for-first-consumer-delete", "Thick", "- name: vg-w1\n- name: vg-w2",
		funcs.StorageClassVolumeBindingModeWaitForFirstConsumer, funcs.StorageClassReclaimPolicyDelete)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateStorageClass_20(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	err = funcs.CreateStorageClass(
		ctx, cl,
		"test-lvm-thick-wait-for-first-consumer-retain", "Thick", "- name: vg-w1\n- name: vg-w2",
		funcs.StorageClassVolumeBindingModeWaitForFirstConsumer, funcs.StorageClassReclaimPolicyRetain)
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteStorageClass(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	err = funcs.DeleteStorageClass(ctx, cl, "test-lvm-thick-immediate-delete")
	if err != nil {
		t.Error(err)
	}
}
