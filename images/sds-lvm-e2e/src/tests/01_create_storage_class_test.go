package test

import (
	"context"
	"fmt"
	"sds-lvm-e2e/funcs"
	"testing"
)

func init() {
	fmt.Println("Create LVM VolumeGroup: vg-w1 in the node")
}

func TestStorageClassCreation_17(t *testing.T) {
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

func TestStorageClassCreation_18(t *testing.T) {
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

func TestStorageClassCreation_19(t *testing.T) {
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

func TestStorageClassCreation_20(t *testing.T) {
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
