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
