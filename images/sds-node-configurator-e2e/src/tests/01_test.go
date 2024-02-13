package test

import (
	"context"
	"fmt"
	"sds-node-configurator-e2e/funcs"
	"testing"
)

func Test11(t *testing.T) {
	ctx := context.Background()
	cl, err := NewKubeClient()
	if err != nil {
		t.Error(err)
	}

	devices, err := funcs.GetAPIBlockDevices(ctx, cl)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("%s", devices)
}
