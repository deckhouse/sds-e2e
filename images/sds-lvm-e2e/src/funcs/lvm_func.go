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

package funcs

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func CreatePV(path string) (string, error) {
	args := []string{path}
	cmd := exec.Command("pvcreate", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return cmd.String(), fmt.Errorf("unable to run cmd: %s, err: %w, stderror = %s", cmd.String(), err, stderr.String())
	}
	return cmd.String(), nil
}

func RemovePV(pvNames []string) (string, error) {
	args := []string{"pvremove", strings.Join(pvNames, " ")}
	cmd := exec.Command("pvremove", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return cmd.String(), fmt.Errorf("unable to run cmd: %s, err: %w, stderr, %s", cmd.String(), err, stderr.String())
	}
	return cmd.String(), nil
}

func CreateVGLocal(vgName string, pvNames []string) (string, error) {
	args := []string{vgName, strings.Join(pvNames, " ")}
	cmd := exec.Command("vgcreate", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return cmd.String(), fmt.Errorf("unable to run cmd: %s, err: %w, stderror: %s", cmd.String(), err, stderr.String())
	}

	return cmd.String(), nil
}

func RemoveVG(vgName string) (string, error) {
	args := []string{vgName}
	cmd := exec.Command("vgremove", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return cmd.String(), fmt.Errorf("unable to run cmd: %s, err: %w, stderr: %s", cmd.String(), err, stderr.String())
	}

	return cmd.String(), nil
}

func CreateThinPool(thinPool, size, VGName string) (string, error) {
	args := []string{"-L", size, "-T", fmt.Sprintf("%s/%s", VGName, thinPool)}
	cmd := exec.Command("lvcreate", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return cmd.String(), fmt.Errorf("unable to run cmd: %s, err: %w, stderr: %s", cmd.String(), err, stderr.String())
	}
	return cmd.String(), nil
}

// 600
func CreateThinLogicalVolume(vgName, tpName, lvName, size string) (string, error) {
	args := []string{"-T", fmt.Sprintf("%s/%s", vgName, tpName), "-n", lvName, "-V", size, "-W", "y", "-y"}
	cmd := exec.Command("lvcreate", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return cmd.String(), fmt.Errorf("unable to run cmd: %s, err: %w, stderr: %s", cmd.String(), err, stderr.String())
	}

	return cmd.String(), nil
}

// 600
func CreateThickLogicalVolume(vgName, lvName, size string) (string, error) {
	args := []string{"-n", fmt.Sprintf("%s/%s", vgName, lvName), "-L", size, "-W", "y", "-y"}
	cmd := exec.Command("lvcreate", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return cmd.String(), fmt.Errorf("unable to run cmd: %s, err: %w, stderr: %s", cmd.String(), err, stderr.String())
	}

	return cmd.String(), nil
}

func ExtendLV(size, vgName, lvName string) (string, error) {
	args := []string{"-L", size, fmt.Sprintf("/dev/%s/%s", vgName, lvName)}
	cmd := exec.Command("lvextend", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return cmd.String(), fmt.Errorf("unable to run cmd: %s, err: %w, stderr: %s", cmd.String(), err, stderr.String())
	}

	return cmd.String(), nil
}

func RemoveLV(vgName, lvName string) (string, error) {
	args := []string{fmt.Sprintf("/dev/%s/%s", vgName, lvName), "-y"}
	cmd := exec.Command("lvremove", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return cmd.String(), fmt.Errorf("unable to run cmd: %s, err: %w, stderr: %s", cmd.String(), err, stderr.String())
	}
	return cmd.String(), nil
}
