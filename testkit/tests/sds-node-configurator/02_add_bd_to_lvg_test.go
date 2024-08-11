package sds_node_configurator

import (
	"context"
	"github.com/deckhouse/sds-e2e/funcs"
	"path/filepath"
	"testing"
)

func TestAddBDtoLVG(t *testing.T) {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		t.Error("Kubeclient problem", err)
	}

	extCl, err := funcs.NewKubeClient("")
	if err != nil {
		t.Error("Parent cluster kubeclient problem", err)
	}

}
