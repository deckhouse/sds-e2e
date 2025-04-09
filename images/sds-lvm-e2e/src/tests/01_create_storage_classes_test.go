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
