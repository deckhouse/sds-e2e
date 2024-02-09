package test

import (
	"context"
	"sds-lvm-e2e/funcs"
	"testing"
)

func TestStorageClassCreation(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	err = funcs.CreateStorageClass(ctx, cl, "Thick", "Immediate", "- name: vg-w1\n- name: vg-w2")
	if err != nil {
		t.Error(err)
	}
}
