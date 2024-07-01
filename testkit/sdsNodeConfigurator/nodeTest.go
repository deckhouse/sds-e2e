/*
Copyright 2024 Flant JSC

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

package sdsNodeConfigurator

import (
	"context"
	"fmt"
	"github.com/deckhouse/sds-e2e/funcs"
	"github.com/melbahja/goph"
	"log"
	"path/filepath"
	"strings"
)

func LvmVolumeGroupCreation() {
	log.Printf("LVM VG creation")
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		funcs.LogFatalIfError(err, "kubeclient problem")
	}

	devices, err := funcs.GetAPIBlockDevices(ctx, cl)
	if err != nil {
		funcs.LogFatalIfError(err, "get block devices problem")
	}

	for _, item := range devices {
		log.Printf("%s: LVM VG creation:\n%s", item.Status.NodeName, funcs.CreateLvmVolumeGroup(ctx, cl, item.Status.NodeName, []string{item.ObjectMeta.Name}))
	}

	for _, ip := range []string{funcs.MasterNodeIP, funcs.InstallWorkerNodeIp, funcs.WorkerNode2} {
		log.Printf("LVM VG creation on %s", ip)
		auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
		if err != nil {
			funcs.LogFatalIfError(err, "SSH connecton problem")
		}
		client := funcs.GetSSHClient(ip, "user", auth)
		defer client.Close()

		out, err := client.Run("sudo vgdisplay -C")
		if !strings.Contains(string(out), "data") || !strings.Contains(string(out), "20.00g") || err != nil {
			funcs.LogFatalIfError(err, "vgdisplay -C error")
		}
		log.Printf("%s: vgdisplay -C %s", ip, out)
	}
}

func LvmPartsSizeChange() {
	ctx := context.Background()
	cl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	if err != nil {
		funcs.LogFatalIfError(err, "Kubeclient problem")
	}

	//extCl, err := funcs.NewKubeClient(filepath.Join(funcs.AppTmpPath, funcs.KubeConfigName))
	//if err != nil {
	//	funcs.LogFatalIfError(err, "Parent cluster kubeclient problem")
	//}

	log.Printf("LVM size change")

	lmvvgs, err := funcs.GetLvmVolumeGroups(ctx, cl)

	for nodeName, LVMVG := range lmvvgs {
		if LVMVG.Status.Nodes[0].Devices[0].PVSize.String() != "20Gi" || LVMVG.Status.Nodes[0].Devices[0].DevSize.String() != "20975192Ki" {
			funcs.LogFatalIfError(nil, fmt.Sprintf("node name: %s, problem with size: %s, %s", nodeName, LVMVG.Status.Nodes[0].Devices[0].PVSize.String(), LVMVG.Status.Nodes[0].Devices[0].DevSize.String()))
		} else {
			fmt.Printf("node name: %s, size ok: %s, %s\n", nodeName, LVMVG.Status.Nodes[0].Devices[0].PVSize.String(), LVMVG.Status.Nodes[0].Devices[0].DevSize.String())
		}
	}

	for _, ip := range []string{funcs.MasterNodeIP, funcs.InstallWorkerNodeIp, funcs.WorkerNode2} {
		log.Printf("LVM size change on %s", ip)
		auth, err := goph.Key(filepath.Join(funcs.AppTmpPath, funcs.PrivKeyName), "")
		if err != nil {
			funcs.LogFatalIfError(err, "SSH connection problem")
		}
		client := funcs.GetSSHClient(ip, "user", auth)
		defer client.Close()
		out, err := client.Run("sudo vgs")
		if !strings.Contains(string(out), "data") || !strings.Contains(string(out), "20.00g") || err != nil {
			funcs.LogFatalIfError(err, fmt.Sprintf("vgs error: %s", out))
		}
		log.Printf("%s: pvs\n%s", ip, out)

		out, err = client.Run("sudo vgdisplay -C")
		if !strings.Contains(string(out), "data") || !strings.Contains(string(out), "20.00g") || err != nil {
			funcs.LogFatalIfError(err, fmt.Sprintf("vgdisplay -C error: %s", out))

		}
		log.Printf("%s: vgdisplay -C\n%s", ip, out)

		out, err = client.Run("sudo lsblk")
		if !strings.Contains(string(out), "sdc") || !strings.Contains(string(out), "20G") || err != nil {
			funcs.LogFatalIfError(err, fmt.Sprintf("lsblk error: %s", out))
		}
		log.Printf("%s: lsblk \n%s", ip, out)

		out, err = client.Run("sudo pvs")
		if !strings.Contains(string(out), "/dev/sdc") || !strings.Contains(string(out), "20G") || err != nil {
			funcs.LogFatalIfError(err, fmt.Sprintf("pvs error: %s", out))
		}
		log.Printf("%s: pvs \n%s", ip, out)
	}
}
